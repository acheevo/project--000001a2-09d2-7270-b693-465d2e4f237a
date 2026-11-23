package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logging(logger *slog.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		requestID, _ := c.Get("request_id")

		logger.Info("HTTP request",
			"method", method,
			"path", path,
			"status", status,
			"latency", latency,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"request_id", requestID,
		)
	})
}
