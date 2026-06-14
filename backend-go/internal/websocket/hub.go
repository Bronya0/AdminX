// Package websocket 实现基于 Hub 模式的实时日志/事件推送。
//
// 集群下通过 Redis pub/sub 跨实例广播:
//   - 任意实例调用 BroadcastLog() → 发布到 Redis channel "ws:logs"
//   - 每个实例的 Hub 订阅该 channel → 收到后转发给本地所有 WebSocket 客户端
//   - 无 Redis 时降级为单实例广播（InProcess）
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// 日志/事件广播的 Redis channel。
const channelLogs = "ws:logs"

// Message 推送给客户端的消息体（对齐 Django LogConsumer: {line, level}）。
type Message struct {
	Line  string `json:"line"`
	Level string `json:"level"`
}

// Client 单个 WebSocket 连接。
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan Message
	closed  bool // 防止 send channel 被重复 close 导致 panic
	closeMu sync.Mutex
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{hub: hub, conn: conn, send: make(chan Message, 64)}
}

// closeSend 安全关闭 send channel（幂等，多次调用不 panic）。
func (c *Client) closeSend() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// readPump 从连接读消息（客户端通常不发送，这里只消费/检测断开）。
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump 把 send channel 的消息写给连接。
func (c *Client) writePump() {
	defer func() {
		_ = c.conn.Close()
		// 兜底恢复：防止向已 close 的 send 写入导致 panic
		recover()
	}()
	for msg := range c.send {
		data, _ := json.Marshal(msg)
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

// Hub 管理所有 WebSocket 客户端 + Redis 订阅。
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	mu         sync.RWMutex
	rdb        *redis.Client
	logger     *slog.Logger
}

// NewHub 构造 Hub。
func NewHub(rdb *redis.Client, logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 256),
		rdb:        rdb,
		logger:     logger,
	}
}

// Run 启动 Hub 主循环（应在单独 goroutine 中运行）。
// 同时订阅 Redis channel 实现跨实例广播。
func (h *Hub) Run(ctx context.Context) {
	// Redis 订阅（集群跨实例广播）
	if h.rdb != nil {
		go h.subscribeRedis(ctx)
	}

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closeSend()
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			// 必须用写锁：default 分支会 delete map，RLock 下并发 delete 会触发
			// Go runtime fatal "concurrent map writes"（无法 recover，进程崩溃）
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					// 发送缓冲满，关闭慢客户端
					client.closeSend()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// HandleConn 处理新 WebSocket 连接（由 gin handler 调用）。
func (h *Hub) HandleConn(conn *websocket.Conn) {
	client := newClient(h, conn)
	h.register <- client

	go client.writePump()
	client.readPump() // 阻塞直到连接关闭
}

// BroadcastLog 广播一条日志到所有客户端（先发 Redis，再由本地订阅触发本地广播）。
// 无 Redis 时直接本地广播。使用非阻塞发送，避免慢消费者拖垮日志生产方。
func (h *Hub) BroadcastLog(line, level string) {
	msg := Message{Line: line, Level: level}

	if h.rdb != nil {
		// 发布到 Redis，所有实例的订阅者都会收到
		data, _ := json.Marshal(msg)
		if err := h.rdb.Publish(context.Background(), channelLogs, data).Err(); err != nil {
			h.logger.Warn("Redis 发布日志失败，降级本地广播", "error", err)
			h.tryLocalBroadcast(msg)
		}
		return
	}
	// 无 Redis: 直接本地广播
	h.tryLocalBroadcast(msg)
}

// tryLocalBroadcast 非阻塞地把消息塞入 broadcast channel。
// channel 满（订阅/客户端消费不过来）时丢弃该条日志并记 warn，
// 避免阻塞调用方（日志生产链路通常不应被 UI 推送拖住）。
func (h *Hub) tryLocalBroadcast(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("日志广播 channel 已满，丢弃一条日志消息")
	}
}

// subscribeRedis 订阅 Redis channel，收到消息后本地广播。
func (h *Hub) subscribeRedis(ctx context.Context) {
	sub := h.rdb.Subscribe(ctx, channelLogs)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var m Message
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				continue
			}
			h.broadcast <- m
		case <-ctx.Done():
			return
		}
	}
}
