// Package captcha 提供验证码 HTTP 接口。
package captcha

import (
	"github.com/freeDog-wy/go-backend-template/internal/handler"
	"github.com/freeDog-wy/go-backend-template/pkg/captcha"

	"github.com/gin-gonic/gin"
)

// Handler 验证码 HTTP 处理器。
type Handler struct {
	gen captcha.Generator
}

// New 创建验证码处理器。
func New(gen captcha.Generator) *Handler {
	return &Handler{gen: gen}
}

// RegisterRoutes 注册路由。
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/captcha", h.Generate)
}

// Generate 生成验证码，返回 id 和 Base64 图片。
func (h *Handler) Generate(c *gin.Context) {
	id, b64img, err := h.gen.Generate()
	if err != nil {
		handler.Fail(c, "CAPTCHA_GENERATE_FAILED", "failed to generate captcha")
		return
	}
	handler.OK(c, captchaResponse{
		CaptchaID: id,
		Image:     b64img,
	})
}
