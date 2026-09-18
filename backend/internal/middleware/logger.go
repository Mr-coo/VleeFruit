package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger emits one structured record per HTTP request. The level follows the
// response status: 5xx -> error, 4xx -> warn, otherwise info. The record
// inherits request_id from the request context (see RequestID).
func Logger(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}

		ctx := c.Request.Context()
		switch {
		case status >= http.StatusInternalServerError:
			l.ErrorContext(ctx, "http request", attrs...)
		case status >= http.StatusBadRequest:
			l.WarnContext(ctx, "http request", attrs...)
		default:
			l.InfoContext(ctx, "http request", attrs...)
		}
	}
}

// Recovery logs any panic through the structured logger and returns a generic
// 500, replacing Gin's default text recovery so panics are captured as records.
func Recovery(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				l.ErrorContext(c.Request.Context(), "panic recovered",
					slog.Any("error", err),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			}
		}()
		c.Next()
	}
}
