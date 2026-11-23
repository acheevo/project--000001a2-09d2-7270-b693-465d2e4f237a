package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	ServiceContextKey   = "service"
	errorKey            = "error"
)

func JWTAuth(jwtService service.JWTService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			logger.Warn("missing authorization header", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "authorization header required"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			logger.Warn("invalid authorization format", "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "bearer token required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			logger.Warn("invalid token attempt", "error", err, "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid token"})
			c.Abort()
			return
		}

		c.Set(ServiceContextKey, claims)
		c.Next()
	}
}

func RequireService(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceCtx, exists := c.Get(ServiceContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "authentication required"})
			c.Abort()
			return
		}

		claims, ok := serviceCtx.(*service.Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: "invalid service context"})
			c.Abort()
			return
		}

		if claims.ServiceName != serviceName {
			c.JSON(http.StatusForbidden, gin.H{errorKey: "service not authorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}
