// Package middleware holds Gin middleware used across the REST API.
package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestID attaches a request id to each request (reusing an inbound one if
// present) and echoes it back in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			buf := make([]byte, 8)
			_, _ = rand.Read(buf)
			id = hex.EncodeToString(buf)
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}
