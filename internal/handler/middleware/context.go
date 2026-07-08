package middleware

import "github.com/gin-gonic/gin"

const CurrentUserIDKey = "current_user_id"

func CurrentUserID(c *gin.Context) uint {
	return c.GetUint(CurrentUserIDKey)
}
