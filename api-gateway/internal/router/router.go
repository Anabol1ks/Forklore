package router

import (
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func Setup(log *zap.Logger, authHandler *handlers.AuthHandler, repositoryHandler *handlers.RepositoryHandler, contentHandler *handlers.ContentHandler, searchHandler *handlers.SearchHandler, studyHandler *handlers.StudyHandler, profileHandler *handlers.ProfileHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Static("/uploads", "./uploads")
	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		// Public read routes with optional auth header (service enforces private visibility checks)
		v1.GET("/repositories/tags", repositoryHandler.ListRepositoryTags)
		v1.GET("/repositories/:repo_id", repositoryHandler.GetRepositoryByID)
		v1.GET("/repositories/:repo_id/forks", repositoryHandler.ListForks)
		v1.GET("/repositories/:repo_id/star", repositoryHandler.GetRepositoryStarState)
		v1.GET("/repositories/:repo_id/documents", contentHandler.ListRepositoryDocuments)
		v1.GET("/repositories/:repo_id/files", contentHandler.ListRepositoryFiles)

		v1.GET("/documents/:document_id", contentHandler.GetDocument)
		v1.GET("/documents/:document_id/versions", contentHandler.ListDocumentVersions)

		v1.GET("/files/:file_id", contentHandler.GetFile)
		v1.GET("/files/:file_id/content", contentHandler.GetFileContent)
		v1.GET("/files/:file_id/versions", contentHandler.ListFileVersions)

		v1.GET("/document-versions/:version_id", contentHandler.GetDocumentVersion)
		v1.GET("/file-versions/:version_id", contentHandler.GetFileVersion)
		v1.POST("/search", searchHandler.Search)
		v1.POST("/study/generate-text", studyHandler.GenerateText)
		v1.POST("/search/users", searchHandler.SearchUsers)

		v1.GET("/users/:owner_id/repositories", repositoryHandler.ListUserRepositories)
		v1.GET("/users/:owner_id/repositories/:slug", repositoryHandler.GetRepositoryBySlug)
		v1.GET("/profiles/by-user/:user_id", profileHandler.GetProfileByUserID)
		v1.GET("/profiles/by-username/:username", profileHandler.GetProfileByUsername)
		v1.GET("/profiles/:user_id/social-links", profileHandler.ListProfileSocialLinks)
		v1.GET("/profiles/:user_id/followers", profileHandler.ListFollowers)
		v1.GET("/profiles/:user_id/following", profileHandler.ListFollowing)
		v1.GET("/profiles/titles", profileHandler.ListAvailableTitles)

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

			// Special paths
			repositories.GET("/me", repositoryHandler.ListMyRepositories)
			repositories.GET("/me/starred", repositoryHandler.ListMyStarredRepositories)

			// Repository write paths (by ID)
			repositories.PATCH("/:repo_id", repositoryHandler.UpdateRepository)
			repositories.DELETE("/:repo_id", repositoryHandler.DeleteRepository)
			repositories.POST("/:repo_id/fork", repositoryHandler.ForkRepository)
			repositories.POST("/:repo_id/star", repositoryHandler.ToggleRepositoryStar)

			// ── Documents ──
			repositories.POST("/:repo_id/documents", contentHandler.CreateDocument)

			// ── Files ──
			repositories.POST("/:repo_id/files", contentHandler.CreateFile)
			repositories.POST("/:repo_id/files/upload", contentHandler.UploadFile)
		}

		// ── Documents ──
		documents := v1.Group("/documents")
		documents.Use(middleware.AuthRequired())
		{
			documents.PATCH("/:document_id/draft", contentHandler.SaveDocumentDraft)
			documents.DELETE("/:document_id", contentHandler.DeleteDocument)

			// ── Document Versions ──
			documents.POST("/:document_id/versions", contentHandler.CreateDocumentVersion)
			documents.POST("/:document_id/versions/:version_id/restore", contentHandler.RestoreDocumentVersion)
		}

		// ── Files ──
		files := v1.Group("/files")
		files.Use(middleware.AuthRequired())
		{
			files.DELETE("/:file_id", contentHandler.DeleteFile)

			// ── File Versions ──
			files.POST("/:file_id/versions", contentHandler.AddFileVersion)
			files.POST("/:file_id/versions/:version_id/restore", contentHandler.RestoreFileVersion)
		}

		search := v1.Group("/search")
		search.Use(middleware.AuthRequired())
		{
			search.POST("/index/repositories", searchHandler.UpsertRepositoryIndex)
			search.DELETE("/index/repositories/:repo_id", searchHandler.DeleteRepositoryIndex)

			search.POST("/index/documents", searchHandler.UpsertDocumentIndex)
			search.DELETE("/index/documents/:document_id", searchHandler.DeleteDocumentIndex)

			search.POST("/index/files", searchHandler.UpsertFileIndex)
			search.DELETE("/index/files/:file_id", searchHandler.DeleteFileIndex)
		}

		profiles := v1.Group("/profiles")
		profiles.Use(middleware.AuthRequired())
		{
			profiles.GET("/me", profileHandler.GetMyProfile)
			profiles.PATCH("/me", profileHandler.UpdateProfile)
			profiles.PATCH("/me/readme", profileHandler.UpdateProfileReadme)
			profiles.PUT("/me/title", profileHandler.SetProfileTitle)
			profiles.POST("/me/image", profileHandler.UploadMyProfileImage)

			profiles.POST("/social-links", profileHandler.UpsertProfileSocialLink)
			profiles.PUT("/social-links", profileHandler.UpsertProfileSocialLink)
			profiles.DELETE("/social-links/:social_link_id", profileHandler.DeleteProfileSocialLink)

			profiles.POST("/:followee_id/follow", profileHandler.FollowUser)
			profiles.DELETE("/:followee_id/follow", profileHandler.UnfollowUser)
		}
	}

	return r
}
