package service

import (
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

// NodesOverview 统计节点总数/在线/离线 + 节点列表（节点管理页顶部卡片）。
func (s *ClusterService) NodesOverview() (*NodesOverview, error) {
	counts, err := s.repo.CountNodesByStatus()
	if err != nil {
		return nil, apperr.ErrInternal
	}
	nodes, err := s.repo.ListAllNodes()
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
		Nodes:       nodes,
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
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     *int   `json:"port"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Version  *string `json:"version"`
	IsActive *bool  `json:"is_active"`
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
	// 已存在，更新
	existing.Name = in.Name
	existing.Version = in.Version
	existing.Host = in.Host
	existing.Description = in.Description
	if in.ExtraInfo != nil {
		existing.ExtraInfo = mustJSONMap(in.ExtraInfo)
	}
	existing.LastHeartbeat = &now
	if err := s.repo.UpdateComponent(existing); err != nil {
		return nil, apperr.ErrInternal
	}
	return existing, nil
}

// Heartbeat 业务组件心跳。
func (s *ClusterService) Heartbeat(appLabel string) (*model.ServiceComponent, error) {
	c, err := s.repo.FindComponentByLabel(appLabel)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(404, "组件未注册: "+appLabel)
		}
		return nil, apperr.ErrInternal
	}
	now := time.Now()
	c.LastHeartbeat = &now
	if err := s.repo.UpdateComponent(c); err != nil {
		return nil, apperr.ErrInternal
	}
	return c, nil
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
	var c model.ServiceComponent
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return apperr.ErrNotFound
	}
	c.UpgradeVersion = version
	c.UpgradeURL = url
	c.UpgradeChecksum = checksum
	return s.repo.UpdateComponent(&c)
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
	var c model.ServiceComponent
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return apperr.ErrNotFound
	}
	c.UpgradeVersion = ""
	c.UpgradeURL = ""
	c.UpgradeChecksum = ""
	return s.repo.UpdateComponent(&c)
}

func (s *ClusterService) SetUninstall(id string) error {
	return s.db.Model(&model.ServiceComponent{}).Where("id = ?", id).
		Update("uninstall_pending", true).Error
}

func (s *ClusterService) CancelUninstall(id string) error {
	return s.db.Model(&model.ServiceComponent{}).Where("id = ?", id).
		Update("uninstall_pending", false).Error
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
