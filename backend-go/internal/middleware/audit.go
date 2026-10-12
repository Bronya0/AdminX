// Package middleware — 声明式操作审计中间件。
//
// 审计此前靠各 handler 手动调 auditRecord，只有用户/角色/菜单三类接上了
// （其余 handler 连 AuditService 依赖都没有），新增模块必然继续漏：
// 站点信息（含验证码开关）/主题/配置中心/定时任务/集群/密码策略/文件都漏了。
// 这里把"哪个写端点产生哪条审计"声明到路由表上，漏没漏一眼可查。
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// 审计动作词表：与前端「操作审计」页的操作类型下拉保持一致（改这里要同步前端）。
const (
	AuditCreate  = "create"
	AuditUpdate  = "update"
	AuditDelete  = "delete"
	AuditExecute = "execute" // 触发类操作：手动执行任务、重载调度器、下发升级/卸载指令
)

// auditSuccessCodes 判定写操作成功的业务码（项目约定：HTTP 恒 200，业务状态在 code）。
var auditSuccessCodes = map[int]struct{}{200: {}, 201: {}, 204: {}}

// AuditRecorder 审计写入接口。定义在中间件侧而非直接用 *service.AuditService：
// service 包的测试反向依赖 middleware，中间件不能反向 import service（会成环）。
type AuditRecorder interface {
	Record(action, modelName, objectID, objectRepr, operator, ip string,
		oldVals, newVals datatypes.JSON, diff string) error
}

// Audited 声明式审计：标注"这个写端点成功时产生一条审计记录"。
//
// 使用约束：
//   - 必须注册在鉴权中间件之后（依赖 context 里的 username）；
//   - 已由 handler 自行记录、带 old/new 快照的路由（用户/角色/菜单写操作）
//     不要再挂它，否则一次操作写两条记录；
//   - 只记成功（code 200/201/204）；失败、403、panic 都不落审计；
//   - 变更摘要（diff_summary）取请求体的**字段名**，不记值：改密、建 webhook 的 body
//     含明文密码与密钥，只记字段名就不会因为漏配脱敏名单而泄密。
func Audited(rec AuditRecorder, action, model string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rec == nil {
			c.Next()
			return
		}

		// 包一层 ResponseWriter 缓存响应体：object_id/object_repr 只能从 handler
		// 的返回值里取（create 的对象 ID 在进 handler 前还不存在）。
		capture := &bodyCapture{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = capture
		reqBody := readJSONBody(c)

		c.Next()

		if !capture.succeeded() {
			return
		}

		operator := c.GetString("username")
		objectID, objectRepr := auditObject(c, capture.buf.Bytes(), operator)
		if err := rec.Record(action, model, objectID, objectRepr, operator,
			c.ClientIP(), nil, nil, auditDiffSummary(reqBody)); err != nil {
			// 审计失败不阻断业务，但必须留痕：静默丢审计等于没有审计
			slog.Default().Warn("操作审计写入失败",
				"action", action, "model", model, "object_id", objectID, "error", err)
		}
	}
}

// bodyCapture 缓存响应体的 ResponseWriter 包装。
// 只覆盖 Write/WriteString（gin 渲染 JSON 走 Write），其余方法由内嵌委托给原 writer。
type bodyCapture struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyCapture) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyCapture) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// succeeded 本次写操作是否成功：HTTP 2xx 且业务 code 属于成功码。
func (w *bodyCapture) succeeded() bool {
	if w.Status() < 200 || w.Status() >= 300 {
		return false
	}
	body := w.buf.Bytes()
	if len(body) == 0 {
		// 无响应体（如 204 No Content）视为成功
		return true
	}
	var env struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return false
	}
	_, ok := auditSuccessCodes[env.Code]
	return ok
}

// auditObject 推导审计对象。
// 优先取响应体 data 里的 id 与展示字段（create/update 会回带对象），
// 其次取路由 :id（delete 只回空 data），都没有时回落到操作者
// （改密、导入这类没有单一对象的操作）。
func auditObject(c *gin.Context, body []byte, operator string) (string, string) {
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	var objectID, objectRepr string
	if err := json.Unmarshal(body, &env); err == nil && env.Data != nil {
		objectID = firstStringField(env.Data, "id")
		objectRepr = firstStringField(env.Data, "name", "username", "code", "title", "key")
	}
	if objectID == "" {
		objectID = c.Param("id")
	}
	if objectRepr == "" {
		objectRepr = objectID
	}
	if objectRepr == "" {
		objectRepr = operator
	}
	// 列宽 object_id=128 / object_repr=255：超长在 PG 上会直接写失败（审计被静默丢掉）
	return truncateRunes(objectID, 128), truncateRunes(objectRepr, 255)
}

// maxAuditBodyBytes 审计解析请求体的上限；超过就不解析（变更摘要留空）。
const maxAuditBodyBytes = 64 << 10

// readJSONBody 读取并复原 JSON 请求体（只用于提取字段名，值一律不落库）。
// 非 JSON（如文件上传的 multipart）直接跳过，不把整个上传缓冲进内存。
func readJSONBody(c *gin.Context) []byte {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") || c.Request.Body == nil {
		return nil
	}
	original := c.Request.Body
	raw, err := io.ReadAll(io.LimitReader(original, maxAuditBodyBytes+1))
	if len(raw) > maxAuditBodyBytes {
		// 超限时不能截断 handler 的输入：已读部分与剩余流拼回去，摘要留空
		c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), original))
		return nil
	}
	// 必须复原请求体：handler 还要自己读一遍
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	return raw
}

// auditDiffSummary 用请求体顶层字段名拼变更摘要（值不落库）。
func auditDiffSummary(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(body, &fields); err != nil {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// firstStringField 按给定顺序取 data 中第一个非空字符串字段。
func firstStringField(data map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := data[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// truncateRunes 按字符（而非字节）截断，避免把多字节字符切成半个。
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
