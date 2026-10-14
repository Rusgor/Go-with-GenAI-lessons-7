package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"homework/internal/model"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "role"
)

type TokenClaims struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user model.User, secret string) (string, error) {
	if len([]byte(secret)) < 32 {
		return "", jwt.ErrInvalidKey
	}
	claims := TokenClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "user:" + strconv.FormatUint(user.ID, 10),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// AuthMiddleware loads the signing secret only from JWT_SECRET when requests arrive.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("JWT_SECRET")
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len([]byte(secret)) < 32 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "authentication required"}})
			return
		}

		claims := &TokenClaims{}
		token, err := jwt.ParseWithClaims(strings.TrimSpace(parts[1]), claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || token == nil || !token.Valid || claims.UserID == 0 || !validRole(claims.Role) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "authentication required"}})
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

func validRole(role string) bool {
	return role == model.UserReader || role == model.UserLibrarian || role == model.UserAdmin
}
