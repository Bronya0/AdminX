package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/response"

	"adminx/internal/captcha"
)

// CaptchaHandler 验证码 HTTP 处理器。
type CaptchaHandler struct {
	mgr    *captcha.Manager
	logger *slog.Logger
}

func NewCaptchaHandler(mgr *captcha.Manager, logger *slog.Logger) *CaptchaHandler {
	return &CaptchaHandler{mgr: mgr, logger: logger}
}

// Generate GET /captcha/captcha/
// 返回 image 为 data:image/png;base64,... 的完整 data URI（不再返回 SVG：
// SVG 是文本，答案就写在节点里，抓一次响应即可读出）。
func (h *CaptchaHandler) Generate(c *gin.Context) {
	captchaID, image, err := h.mgr.Generate()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, gin.H{
		"captcha_id": captchaID,
		"image":      image,
	})
}

// Verify POST /captcha/captcha/verify/
func (h *CaptchaHandler) Verify(c *gin.Context) {
	var req struct {
		CaptchaID   string `json:"captcha_id" binding:"required"`
		CaptchaText string `json:"captcha_text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	ok, err := h.mgr.Verify(req.CaptchaID, req.CaptchaText)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	if !ok {
		response.Fail(c, 400, "验证码错误或已过期")
		return
	}
	response.OK(c, gin.H{"valid": true})
}
