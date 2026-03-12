package router

import (
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func Setup(log *zap.Logger, authHandler *handlers.AuthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(log))

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		// ── Auth ──
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)

			protected := auth.Group("")
			protected.Use(middleware.AuthRequired())
			{
				protected.POST("/logout-all", authHandler.LogoutAll)
				protected.GET("/me", authHandler.GetMe)
			}
		}

		// ── Future microservices ──
		// projects := v1.Group("/projects")
		// { ... }
	}

	return r
}
