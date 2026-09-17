package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminAuth guards management endpoints with a static bearer token. This is a
// placeholder for a fuller auth story (JWT/users) noted as out of scope.
func AdminAuth(token string) gin.HandlerFunc {
	want := []byte(token)
	return func(c *gin.Context) {
		got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
