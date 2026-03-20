package middleware

import (
	"net/http"
	"strings"

	"api-gateway/internal/models"

	"github.com/gin-gonic/gin"
)

// AuthRequired проверяет наличие Bearer токена в заголовке Authorization.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "authorization header with Bearer token required",
			})
			return
		}

		c.Next()
	}
}
