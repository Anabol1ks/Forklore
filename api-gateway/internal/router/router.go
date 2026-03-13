package router

import (
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func Setup(log *zap.Logger, authHandler *handlers.AuthHandler, repositoryHandler *handlers.RepositoryHandler) *gin.Engine {
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

		// ── Repositories ──
		repositories := v1.Group("/repositories")
		repositories.Use(middleware.AuthRequired())
		{
			// Create repository
			repositories.POST("", repositoryHandler.CreateRepository)

			// Special paths (must be before parameterized paths)
			repositories.GET("/me", repositoryHandler.ListMyRepositories)
			repositories.GET("/tags", repositoryHandler.ListRepositoryTags)

			// Repository-specific paths (by ID)
			repositories.GET("/:repo_id", repositoryHandler.GetRepositoryByID)
			repositories.PATCH("/:repo_id", repositoryHandler.UpdateRepository)
			repositories.DELETE("/:repo_id", repositoryHandler.DeleteRepository)
			repositories.POST("/:repo_id/fork", repositoryHandler.ForkRepository)
			repositories.GET("/:repo_id/forks", repositoryHandler.ListForks)
		}

		// ── Users - for user-specific repository operations ──
		users := v1.Group("/users")
		users.Use(middleware.AuthRequired())
		{
			// Get user repositories
			users.GET("/:owner_id/repositories", repositoryHandler.ListUserRepositories)

			// Get repository by owner and slug
			users.GET("/:owner_id/repositories/:slug", repositoryHandler.GetRepositoryBySlug)
		}
	}

	return r
}
