package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps the number of bytes handlers will read from a request body.
// It wraps the body in an http.MaxBytesReader, so an oversized body makes the
// bind fail (413) instead of letting a client stream unbounded data into memory.
// A max of 0 or less disables the limit.
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if max > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}
