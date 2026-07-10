package middleware

import (
	"errors"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/handler"
	svcAuth "github.com/freeDog-wy/go-backend-template/internal/usecase/auth"

	"github.com/gin-gonic/gin"
)

func RequireAuth(authSvc *svcAuth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authenticateRequest(c, authSvc)
		if !ok {
			return
		}
		c.Set(CurrentUserIDKey, userID)
		c.Next()
	}
}

func authenticateRequest(c *gin.Context, authSvc *svcAuth.Service) (uint, bool) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(authHeader) < len("Bearer ")+1 || !strings.EqualFold(authHeader[:len("Bearer ")], "Bearer ") {
		handler.Fail(c, "UNAUTHORIZED", "missing access token")
		c.Abort()
		return 0, false
	}

	token := strings.TrimSpace(authHeader[len("Bearer "):])
	identity, err := authSvc.AuthenticateAccessToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, svcAuth.ErrInvalidAccessToken) {
			handler.Fail(c, "UNAUTHORIZED", "access token is invalid or expired")
		} else {
			handler.Fail(c, "INTERNAL_ERROR", err.Error())
		}
		c.Abort()
		return 0, false
	}

	return identity.UserID, true
}
