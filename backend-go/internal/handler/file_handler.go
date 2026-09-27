package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/service"
)

// FileHandler 文件中心 HTTP 处理器。
type FileHandler struct {
	svc    *service.FileService
	logger *slog.Logger
}

func NewFileHandler(svc *service.FileService, logger *slog.Logger) *FileHandler {
	return &FileHandler{svc: svc, logger: logger}
}

// Upload POST /files/upload/ (multipart 字段名: file)
//
// 直接消费 multipart part 流式落盘。不用 c.FormFile：它会把整个请求体先写进
// 系统临时目录，用它可以绕过大小上限把 /tmp 写满。
func (h *FileHandler) Upload(c *gin.Context) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		response.Fail(c, 400, "请求必须是 multipart/form-data")
		return
	}
	part, err := nextFilePart(mr)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	defer func() { _ = part.Close() }()

	// uploadedBy 用 user_id（归属校验与审计追溯需要唯一标识）
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	record, err := h.svc.Upload(part.FileName(), part, uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, record)
}

// nextFilePart 顺序读取表单，返回名为 file 的文件字段（跳过其他普通字段）。
func nextFilePart(mr *multipart.Reader) (*multipart.Part, error) {
	for {
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("未提供文件")
			}
			return nil, fmt.Errorf("解析上传表单失败: %w", err)
		}
		if part.FormName() == "file" && part.FileName() != "" {
			return part, nil
		}
		_ = part.Close()
	}
}

// Records GET /files/records/
func (h *FileHandler) Records(c *gin.Context) {
	p := pagination.Parse(c)
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	items, count, err := h.svc.List(p.Offset(), p.Size, uid, isSuperuser(c))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, items)
}

// Download GET /files/records/:id/download/
// 唯一取文件的入口（需 JWT + files 权限，且非超管只能下自己的文件）。
func (h *FileHandler) Download(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	fullPath, originalName, err := h.svc.DownloadPath(c.Param("id"), uid, isSuperuser(c))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	// 文件流原样返回（下载接口不套统一 JSON 响应壳）
	c.FileAttachment(fullPath, originalName)
}

// Delete DELETE /files/records/:id/
func (h *FileHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	if err := h.svc.Delete(c.Param("id"), uid, isSuperuser(c)); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}
