package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, authenticated := c.Get(ContextRole)
		role, isString := roleValue.(string)
		if !authenticated || !isString || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "authentication required"}})
			return
		}
		for _, candidate := range allowed {
			if role == candidate {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "insufficient permissions"}})
	}
}
