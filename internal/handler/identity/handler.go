package identity

import (
	"github.com/freeDog-wy/go-backend-template/internal/handler"

	"github.com/gin-gonic/gin"
)

// Handler 身份域 HTTP 处理器。
type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(route *gin.Engine) {
	group := route.Group("/api/v1")
	{
		group.GET("/users", h.GetUsers)
	}
}

func (h *Handler) GetUsers(c *gin.Context) {
	handler.OK(c, []string{})
}
