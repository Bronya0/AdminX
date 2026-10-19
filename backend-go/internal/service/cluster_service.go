package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// ClusterService 集群管理业务。
type ClusterService struct {
	db   *gorm.DB
	repo *repository.ClusterRepo
}

func NewClusterService(db *gorm.DB, repo *repository.ClusterRepo) *ClusterService {
	return &ClusterService{db: db, repo: repo}
}

// ListNodes 节点列表（支持 search/status 过滤）。
func (s *ClusterService) ListNodes(offset, limit int, search, status string) ([]model.ClusterNode, int64, error) {
	return s.repo.ListNodes(offset, limit, search, status)
}

// GetNode 按 ID 查询节点。
func (s *ClusterService) GetNode(id string) (*model.ClusterNode, error) {
	n, err := s.repo.FindNodeByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return n, nil
}

// NodesOverview 节点概览（对齐前端 types 的 ClusterOverview）。
type NodesOverview struct {
	Total       int64               `json:"total"`
	Online      int64               `json:"online"`
	Offline     int64               `json:"offline"`
	Maintenance int64               `json:"maintenance"`
	Nodes       []model.ClusterNode `json:"nodes"`
}

// NodesOverview 只统计节点总数/在线/离线（不返回节点行）。
// Nodes 置空数组保持 JSON 形状兼容（nodes: [] 而非 null），列表请走 ListNodes 分页接口。
func (s *ClusterService) NodesOverview() (*NodesOverview, error) {
	counts, err := s.repo.CountNodesByStatus()
	if err != nil {
		return nil, apperr.ErrInternal
	}
	var total int64
	for _, c := range counts {
		total += c
	}
	return &NodesOverview{
		Total:       total,
		Online:      counts["online"],
		Offline:     counts["offline"],
		Maintenance: counts["maintenance"],
		Nodes:       []model.ClusterNode{},
	}, nil
}

func (s *ClusterService) CreateNode(n *model.ClusterNode) (*model.ClusterNode, error) {
	if err := s.repo.CreateNode(n); err != nil {
		return nil, apperr.ErrInternal
	}
	return n, nil
}

// NodeCreateInput 节点创建入参（白名单：不允许客户端指定 id/心跳/时间戳等系统字段）。
type NodeCreateInput struct {
	Name     string `json:"name" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	IsActive *bool  `json:"is_active"`
}

func (s *ClusterService) CreateNodeInput(in NodeCreateInput) (*model.ClusterNode, error) {
	n := &model.ClusterNode{
		Name:    in.Name,
		Host:    in.Host,
		Port:    in.Port,
		Role:    in.Role,
		Status:  in.Status,
		Version: in.Version,
	}
	if n.Port == 0 {
		n.Port = 8000
	}
	if n.Role == "" {
		n.Role = "slave"
	}
	if n.Status == "" {
		n.Status = "offline"
	}
	if in.IsActive != nil {
		n.IsActive = *in.IsActive
	} else {
		n.IsActive = true
	}
	if err := s.repo.CreateNode(n); err != nil {
		return nil, apperr.ErrInternal
	}
	return n, nil
}

// NodeUpdateInput 节点更新入参（白名单字段，杜绝 map 直通 Updates 的任意列写入）。
type NodeUpdateInput struct {
	Name     string  `json:"name"`
	Host     string  `json:"host"`
	Port     *int    `json:"port"`
	Role     string  `json:"role"`
	Status   string  `json:"status"`
	Version  *string `json:"version"`
	IsActive *bool   `json:"is_active"`
}

func (s *ClusterService) UpdateNode(id string, in NodeUpdateInput) (*model.ClusterNode, error) {
	updates := map[string]interface{}{}
	if in.Name != "" {
		updates["name"] = in.Name
	}
	if in.Host != "" {
		updates["host"] = in.Host
	}
	if in.Port != nil {
		updates["port"] = *in.Port
	}
	if in.Role != "" {
		updates["role"] = in.Role
	}
	if in.Status != "" {
		updates["status"] = in.Status
	}
	if in.Version != nil {
		updates["version"] = *in.Version
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}
	if len(updates) > 0 {
		if err := s.db.Model(&model.ClusterNode{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, apperr.ErrInternal
		}
	}
	var n model.ClusterNode
	if err := s.db.First(&n, "id = ?", id).Error; err != nil {
		return nil, apperr.ErrNotFound
	}
	return &n, nil
}

func (s *ClusterService) DeleteNode(id string) error {
	return s.repo.DeleteNode(id)
}

// ── 业务组件注册/心跳 ──

func (s *ClusterService) ListComponents(offset, limit int) ([]model.ServiceComponent, int64, error) {
	return s.repo.ListComponents(offset, limit)
}

// GetComponent 按 ID 查询业务组件。
func (s *ClusterService) GetComponent(id string) (*model.ServiceComponent, error) {
	c, err := s.repo.FindComponentByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return c, nil
}

// DeleteComponent 删除业务组件注册记录（先确认存在，避免无效删除静默成功）。
func (s *ClusterService) DeleteComponent(id string) error {
	if _, err := s.GetComponent(id); err != nil {
		return err
	}
	if err := s.repo.DeleteComponent(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// ConfirmUpgrade 确认升级已完成：清空升级指令（升级指令已被业务侧消费）。
// 与 CancelUpgrade 同义（都是清除待执行指令），分开命名是为了接口语义清晰。
func (s *ClusterService) ConfirmUpgrade(id string) error {
	return s.CancelUpgrade(id)
}

// RegisterInput 业务组件注册入参。
type RegisterInput struct {
	AppLabel    string                 `json:"app_label" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Version     string                 `json:"version"`
	Host        string                 `json:"host"`
	Description string                 `json:"description"`
	ExtraInfo   map[string]interface{} `json:"extra_info"`
}

// Register 业务组件注册（幂等：存在则更新，不存在则创建）。
func (s *ClusterService) Register(in RegisterInput) (*model.ServiceComponent, error) {
	existing, err := s.repo.FindComponentByLabel(in.AppLabel)
	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 新注册
		c := &model.ServiceComponent{
			AppLabel:    in.AppLabel,
			Name:        in.Name,
			Version:     in.Version,
			Host:        in.Host,
			Description: in.Description,
			ExtraInfo:   mustJSONMap(in.ExtraInfo),
		}
		c.LastHeartbeat = &now
		if err := s.repo.CreateComponent(c); err != nil {
			return nil, apperr.ErrInternal
		}
		return c, nil
	}
	if err != nil {
		return nil, apperr.ErrInternal
	}
	// 已存在：按列更新注册信息 + 心跳时间。
	// 不能读整行再整行写回：那样会把并发下发的升级/卸载指令一并抹成旧值。
	cols := map[string]interface{}{
		"name": in.Name, "version": in.Version, "host": in.Host,
		"description": in.Description, "last_heartbeat": now,
	}
	if in.ExtraInfo != nil {
		cols["extra_info"] = mustJSONMap(in.ExtraInfo)
	}
	if err := s.updateComponentColumns("id = ?", []interface{}{existing.ID}, cols); err != nil {
		return nil, err
	}
	return s.repo.FindComponentByLabel(in.AppLabel)
}

// Heartbeat 业务组件心跳：只写心跳列，再回读整行组装响应。
// 不能读整行改完再整行写回：那样会把并发下发的升级/卸载指令抹掉。
func (s *ClusterService) Heartbeat(appLabel string) (*model.ServiceComponent, error) {
	err := s.updateComponentColumns("app_label = ?", []interface{}{appLabel},
		map[string]interface{}{"last_heartbeat": time.Now()})
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, apperr.New(404, "组件未注册: "+appLabel)
	}
	if err != nil {
		return nil, err
	}
	return s.repo.FindComponentByLabel(appLabel)
}

// SetUpgrade 下发升级指令。
// url + SHA-256 checksum 必填：业务侧据此下载并校验升级包，缺任一项就只会在业务侧
// 被当成非法指令丢弃（成功设置、永不生效），因此在平台侧就拒绝。
func (s *ClusterService) SetUpgrade(id, version, url, checksum string) error {
	version = strings.TrimSpace(version)
	url = strings.TrimSpace(url)
	checksum = strings.TrimSpace(checksum)
	if version == "" || url == "" {
		return apperr.New(400, "升级指令必须包含版本号与下载地址")
	}
	if !isSHA256Hex(checksum) {
		return apperr.New(400, "升级指令必须包含 SHA-256 校验和（64 位十六进制）")
	}
	return s.updateComponentColumns("id = ?", []interface{}{id}, map[string]interface{}{
		"upgrade_version": version, "upgrade_url": url, "upgrade_checksum": checksum,
	})
}

// isSHA256Hex 校验 64 位十六进制 SHA-256 摘要。
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch ch := s[i]; {
		case ch >= '0' && ch <= '9', ch >= 'a' && ch <= 'f', ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
}

func (s *ClusterService) CancelUpgrade(id string) error {
	return s.updateComponentColumns("id = ?", []interface{}{id}, map[string]interface{}{
		"upgrade_version": "", "upgrade_url": "", "upgrade_checksum": "",
	})
}

func (s *ClusterService) SetUninstall(id string) error {
	return s.updateComponentColumns("id = ?", []interface{}{id}, map[string]interface{}{"uninstall_pending": true})
}

func (s *ClusterService) CancelUninstall(id string) error {
	return s.updateComponentColumns("id = ?", []interface{}{id}, map[string]interface{}{"uninstall_pending": false})
}

// updateComponentColumns 按列更新组件。cols 必须由白名单字段构造：
// 结构体更新会跳过零值，而清空升级指令这类“改成空串”的操作必须用 map。
// 没有任何行被更新时返回 ErrNotFound，避免“下发成功但组件不存在”的假象。
func (s *ClusterService) updateComponentColumns(where string, args []interface{}, cols map[string]interface{}) error {
	res := s.db.Model(&model.ServiceComponent{}).Where(where, args...).Updates(cols)
	if res.Error != nil {
		return apperr.ErrInternal
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Unregister 注销组件（按 app_label 删除）。
func (s *ClusterService) Unregister(appLabel string) error {
	c, err := s.repo.FindComponentByLabel(appLabel)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // 未注册视为成功（幂等）
		}
		return apperr.ErrInternal
	}
	if err := s.repo.DeleteComponent(c.ID); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// mustJSONMap map → datatypes.JSON。
func mustJSONMap(m map[string]interface{}) datatypesJSON {
	if m == nil {
		return datatypesJSON([]byte("{}"))
	}
	b, _ := jsonMarshal(m)
	return datatypesJSON(b)
}

func (s *ClusterService) GetComponentByLabel(label string) (*model.ServiceComponent, error) {
	component, err := s.repo.FindComponentByLabel(label)
	if err != nil {
		return nil, err
	}
	return component, nil
}
