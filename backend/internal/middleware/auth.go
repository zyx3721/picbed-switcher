package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/utils"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}

		claims, err := utils.ParseToken(secret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录状态已失效"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Name)
		c.Next()
	}
}

func UserID(c *gin.Context) uint {
	value, ok := c.Get("userID")
	if !ok {
		return 0
	}
	userID, ok := value.(uint)
	if !ok {
		return 0
	}
	return userID
}
