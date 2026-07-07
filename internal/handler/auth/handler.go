package auth

import (
	"errors"

	"github.com/freeDog-wy/go-backend-template/internal/handler"
	svcAuth "github.com/freeDog-wy/go-backend-template/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// Handler 认证 HTTP 处理器。
type Handler struct {
	svc *svcAuth.Service
}

func New(svc *svcAuth.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(route *gin.Engine) {
	group := route.Group("/api/v1")
	{
		// 兼容旧入口，同时补齐 blueprint 中的 auth 路由。
		group.POST("/users/register", h.Register)
		group.POST("/auth/register", h.Register)
		group.POST("/auth/resend-verification", h.ResendVerification)
		group.POST("/auth/verify-email", h.VerifyEmail)
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, "INVALID_INPUT", err.Error())
		return
	}

	result, err := h.svc.Register(c.Request.Context(), req.ToCommand())
	if err != nil {
		switch {
		case errors.Is(err, svcAuth.ErrInvalidCaptcha):
			handler.Fail(c, "INVALID_CAPTCHA", "验证码错误")
		case errors.Is(err, svcAuth.ErrEmailTaken):
			handler.Fail(c, "EMAIL_TAKEN", "邮箱已注册")
		default:
			handler.Fail(c, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	handler.OK(c, FromResult(result))
}

func (h *Handler) ResendVerification(c *gin.Context) {
	var req ResendVerificationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.svc.ResendVerification(c.Request.Context(), req.ToCommand()); err != nil {
		handler.Fail(c, "INTERNAL_ERROR", err.Error())
		return
	}

	handler.OK(c, MessageResponse{Message: "如果账号存在且尚未验证，验证邮件已重新发送"})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.svc.VerifyEmail(c.Request.Context(), req.ToCommand()); err != nil {
		switch {
		case errors.Is(err, svcAuth.ErrInvalidVerificationToken):
			handler.Fail(c, "INVALID_VERIFICATION_TOKEN", "验证链接无效或已过期")
		default:
			handler.Fail(c, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	handler.OK(c, MessageResponse{Message: "邮箱验证成功"})
}
