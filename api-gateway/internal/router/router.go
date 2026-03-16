package router

import (
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func Setup(log *zap.Logger, authHandler *handlers.AuthHandler, repositoryHandler *handlers.RepositoryHandler, contentHandler *handlers.ContentHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
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

			// ── Documents ──
			repositories.POST("/:repo_id/documents", contentHandler.CreateDocument)
			repositories.GET("/:repo_id/documents", contentHandler.ListRepositoryDocuments)

			// ── Files ──
			repositories.POST("/:repo_id/files", contentHandler.CreateFile)
			repositories.GET("/:repo_id/files", contentHandler.ListRepositoryFiles)
		}

		// ── Documents ──
		documents := v1.Group("/documents")
		documents.Use(middleware.AuthRequired())
		{
			documents.GET("/:document_id", contentHandler.GetDocument)
			documents.PATCH("/:document_id/draft", contentHandler.SaveDocumentDraft)
			documents.DELETE("/:document_id", contentHandler.DeleteDocument)

			// ── Document Versions ──
			documents.POST("/:document_id/versions", contentHandler.CreateDocumentVersion)
			documents.GET("/:document_id/versions", contentHandler.ListDocumentVersions)
			documents.POST("/:document_id/versions/:version_id/restore", contentHandler.RestoreDocumentVersion)
		}

		// ── Document Versions (public access) ──
		docVersions := v1.Group("/document-versions")
		{
			docVersions.GET("/:version_id", contentHandler.GetDocumentVersion)
		}

		// ── Files ──
		files := v1.Group("/files")
		files.Use(middleware.AuthRequired())
		{
			files.GET("/:file_id", contentHandler.GetFile)
			files.DELETE("/:file_id", contentHandler.DeleteFile)

			// ── File Versions ──
			files.POST("/:file_id/versions", contentHandler.AddFileVersion)
			files.GET("/:file_id/versions", contentHandler.ListFileVersions)
			files.POST("/:file_id/versions/:version_id/restore", contentHandler.RestoreFileVersion)
		}

		// ── File Versions (public access) ──
		fileVersions := v1.Group("/file-versions")
		{
			fileVersions.GET("/:version_id", contentHandler.GetFileVersion)
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
