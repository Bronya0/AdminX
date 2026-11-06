package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/service"
)

// ConfigHandler 配置中心 HTTP 处理器。
type ConfigHandler struct {
	audit  *service.AuditService
	svc    *service.ConfigService
	logger *slog.Logger
}

func NewConfigHandler(svc *service.ConfigService, audit *service.AuditService, logger *slog.Logger) *ConfigHandler {
	return &ConfigHandler{svc: svc, audit: audit, logger: logger}
}

func (h *ConfigHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	cfgs, count, err := h.svc.List(p.Offset(), p.Size, c.Query("search"), c.Query("group"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, cfgs)
}

func (h *ConfigHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的 ID")
		return
	}
	cfg, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, cfg)
}

func (h *ConfigHandler) Create(c *gin.Context) {
	var in service.ConfigCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	cfg, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "create", "Config", strconv.FormatInt(cfg.ID, 10), cfg.Key, nil, service.ToConfigDTO(cfg))
	response.Created(c, service.ToConfigDTO(cfg))
}

func (h *ConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的 ID")
		return
	}
	var in service.ConfigUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	old, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	cfg, err := h.svc.Update(id, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "update", "Config", strconv.FormatInt(cfg.ID, 10), cfg.Key, old, service.ToConfigDTO(cfg))
	response.OK(c, service.ToConfigDTO(cfg))
}

func (h *ConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的 ID")
		return
	}
	old, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "delete", "Config", strconv.FormatInt(id, 10), old.Key, old, nil)
	response.NoContent(c)
}

// ByGroup GET /config/by_group/?group=xxx
func (h *ConfigHandler) ByGroup(c *gin.Context) {
	group := c.Query("group")
	if group == "" {
		response.Fail(c, 400, "缺少 group 参数")
		return
	}
	result, err := h.svc.GetByGroup(c.Request.Context(), group)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

// GetValue GET /config/get_value/?key=xxx&default=yyy
func (h *ConfigHandler) GetValue(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, 400, "缺少 key 参数")
		return
	}
	defaultVal := c.Query("default")
	val, err := h.svc.GetValue(c.Request.Context(), key, defaultVal)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, val)
}

// Groups GET /config/groups/
func (h *ConfigHandler) Groups(c *gin.Context) {
	groups, err := h.svc.Groups()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, groups)
}

// Export GET /config/export/ — 导出全部非加密字典配置（文件内容由前端下载）。
func (h *ConfigHandler) Export(c *gin.Context) {
	file, err := h.svc.ExportFile()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, file)
}

// Import POST /config/import/ — 按 key 覆盖式导入字典配置。
// 先用 MaxBytesReader 限流 2MB：GetRawData 无上限，大包会把内存打爆。
func (h *ConfigHandler) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	raw, err := c.GetRawData()
	if err != nil {
		response.Fail(c, 400, "读取请求体失败")
		return
	}
	// 审计快照只取本次导入涉及的 key：之前用 List(0, -1) 把整张 configs 表读进内存，
	// 导入文件限了 1000 条，被读的表却没有上限。
	// key 的规范化与 service 侧一致（TrimSpace），否则旧值比对会漏掉带空白的 key。
	var imported service.ConfigFile
	_ = json.Unmarshal(raw, &imported) // 完整格式校验由 ImportFile 统一负责
	keys := make([]string, 0, len(imported.Items))
	for _, item := range imported.Items {
		if key := strings.TrimSpace(item.Key); key != "" {
			keys = append(keys, key)
		}
	}
	// 超限的文件交给 ImportFile 返回「单次最多导入 N 条」的 400：这里若先去查快照，
	// keys 会展开成十几万个 IN 占位符（2MB body 够塞），PG/SQLite 直接报参数上限 → 500。
	// 跳过查询，审计走 snapshot_unavailable 兜底，用户看到的错误提示与原来完全一致。
	var oldByKey map[string]service.ConfigDTO
	if len(keys) <= service.MaxConfigImportItems {
		oldByKey, err = h.svc.FindByKeys(keys)
		if err != nil {
			response.Error(c, h.logger, err)
			return
		}
	}
	result, err := h.svc.ImportFile(c.Request.Context(), raw)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	// 走到这里说明 ImportFile 已校过条数上限，下面这次查询的 keys 必然在上限内。
	newByKey, readErr := h.svc.FindByKeys(keys)
	if readErr != nil {
		h.logger.Error("配置导入后审计快照读取失败", "error", readErr)
		newByKey = nil
	}
	// 审计逐条攒起来，最后一次性批量写入：一次导入最多 1000 条，逐条 INSERT 会拖成 1000 次往返。
	records := make([]service.AuditRecord, 0, len(imported.Items))
	for _, item := range imported.Items {
		key := strings.TrimSpace(item.Key)
		var old interface{}
		action := "create"
		if previous, ok := oldByKey[key]; ok {
			old = previous
			action = "update"
		}
		current, exists := newByKey[key]
		objectID := key
		if exists {
			objectID = strconv.FormatInt(current.ID, 10)
		}
		var updated interface{} = current
		if readErr != nil || !exists {
			updated = gin.H{"snapshot_unavailable": true, "key": key}
		}
		records = append(records, buildAuditRecord(c, action, "Config", objectID, key, "导入配置", old, updated))
	}
	if h.audit != nil {
		if err := h.audit.RecordMany(records); err != nil {
			h.logger.Error("配置导入审计批量写入失败", "count", len(records), "error", err)
		}
	}
	response.OK(c, result)
}
