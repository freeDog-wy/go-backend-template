package user

import (
	"errors"

	"github.com/freeDog-wy/go-backend-template/internal/handler"
	svcUser "github.com/freeDog-wy/go-backend-template/internal/service/user"

	"github.com/gin-gonic/gin"
)

// Handler 用户 HTTP 处理器。
type Handler struct {
	svc *svcUser.Service
}

func New(svc *svcUser.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(route *gin.Engine) {
	group := route.Group("/api/v1")
	{
		group.GET("/users", h.GetUsers)
		group.POST("/users/register", h.Register)
	}
}

func (h *Handler) GetUsers(c *gin.Context) {
	handler.OK(c, []string{})
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
		case errors.Is(err, svcUser.ErrInvalidCaptcha):
			handler.Fail(c, "INVALID_CAPTCHA", "验证码错误")
		case errors.Is(err, svcUser.ErrEmailTaken):
			handler.Fail(c, "EMAIL_TAKEN", "邮箱已注册")
		default:
			handler.Fail(c, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	handler.OK(c, FromResult(result))
}
