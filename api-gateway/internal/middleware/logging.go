package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger возвращает gin middleware для логирования HTTP запросов.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log.Info("HTTP",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("dur", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}
