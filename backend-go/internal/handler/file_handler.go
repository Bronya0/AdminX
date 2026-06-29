package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"djangoadminx/pkg/pagination"
	"djangoadminx/pkg/response"

	"djangoadminx/internal/service"
)

// FileHandler 文件中心 HTTP 处理器。
type FileHandler struct {
	svc    *service.FileService
	logger *slog.Logger
}

func NewFileHandler(svc *service.FileService, logger *slog.Logger) *FileHandler {
	return &FileHandler{svc: svc, logger: logger}
}

// Upload POST /files/upload/ (multipart: "file")
func (h *FileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error("获取上传文件失败", "error", err)
		response.Fail(c, 400, "未提供文件")
		return
	}
	// uploadedBy 用 user_id（审计追溯需要唯一标识）
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	record, err := h.svc.Upload(file, uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, record)
}

// Records GET /files/records/
func (h *FileHandler) Records(c *gin.Context) {
	p := pagination.Parse(c)
	items, count, err := h.svc.List(p.Offset(), p.Size)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, items)
}

// Delete DELETE /files/records/:id/
func (h *FileHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}
