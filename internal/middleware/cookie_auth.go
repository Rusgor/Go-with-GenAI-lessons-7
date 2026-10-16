package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// OptionalCookieAuth reads the JWT from an HttpOnly cookie for server-rendered pages.
// A missing or invalid cookie leaves the request anonymous; later guards decide the response.
func OptionalCookieAuth(cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("JWT_SECRET")
		raw, err := c.Cookie(cookieName)
		if err == nil && raw != "" && len([]byte(secret)) >= 32 {
			if claims, ok := parseClaims(raw, secret); ok {
				c.Set(ContextUserID, claims.UserID)
				c.Set(ContextRole, claims.Role)
			}
		}
		c.Next()
	}
}
