package handlers

import (
	"net/http"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	"github.com/gin-gonic/gin"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
)

type ContentHandler struct {
	client *clients.ContentClient
}

func NewContentHandler(client *clients.ContentClient) *ContentHandler {
	return &ContentHandler{client: client}
}

// CreateDocument godoc
//
//	@Summary		Создать документ
//	@Description	Создаёт новый документ в репозитории
//	@Tags			content
//	@Security		BearerAuth
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Param			body	body	models.CreateDocumentRequest	true	"Данные документа"
//	@Success		201	{object}	models.DocumentResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/repositories/{repo_id}/documents [post]
func (h *ContentHandler) CreateDocument(c *gin.Context) {
	repoID := c.Param("repo_id")

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	protoReq := &contentv1.CreateDocumentRequest{
		RepoId:         &commonv1.UUID{Value: repoID},
		Title:          req.Title,
		Slug:           req.Slug,
		InitialContent: req.InitialContent,
		ChangeSummary:  req.ChangeSummary,
	}

	resp, err := h.client.Client.CreateDocument(ctx, protoReq)
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.DocumentResponse{
		Document: mapDocumentWithDraft(resp.Document, resp.Draft, resp.CurrentVersion),
	})
}

// GetDocument godoc
//
//	@Summary		Получить документ
//	@Description	Получает информацию о документе по ID
//	@Tags			content
//	@Param			document_id	path	string	true	"ID документа"
//	@Success		200	{object}	models.DocumentResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id} [get]
func (h *ContentHandler) GetDocument(c *gin.Context) {
	documentID := c.Param("document_id")

	ctx := forwardAuth(c)
	resp, err := h.client.Client.GetDocumentById(ctx, &contentv1.GetDocumentByIdRequest{
		DocumentId: &commonv1.UUID{Value: documentID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.DocumentResponse{
		Document: mapDocumentWithDraft(resp.Document, resp.Draft, resp.CurrentVersion),
	})
}

// ListRepositoryDocuments godoc
//
//	@Summary		Список документов репозитория
//	@Description	Получает список документов в репозитории с пагинацией
//	@Tags			content
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Param			limit	query	int	false	"Лимит записей"	default(10)
//	@Param			offset	query	int	false	"Смещение"	default(0)
//	@Success		200	{object}	models.DocumentListResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/repositories/{repo_id}/documents [get]
func (h *ContentHandler) ListRepositoryDocuments(c *gin.Context) {
	repoID := c.Param("repo_id")
	limit := int32(10)
	offset := int32(0)

	if l := c.Query("limit"); l != "" {
		var val int32
		if _, err := c.Cookie("limit"); err == nil {
			c.Query("limit")
		}
		limit = int32(val)
	}
	if o := c.Query("offset"); o != "" {
		var val int32
		if _, err := c.Cookie("offset"); err == nil {
			c.Query("offset")
		}
		offset = int32(val)
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.ListRepositoryDocuments(ctx, &contentv1.ListRepositoryDocumentsRequest{
		RepoId: &commonv1.UUID{Value: repoID},
		Limit:  uint32(limit),
		Offset: uint32(offset),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	documents := make([]models.DocumentDetailResponse, len(resp.Documents))
	for i, d := range resp.Documents {
		documents[i] = mapDocument(d)
	}

	c.JSON(http.StatusOK, models.DocumentListResponse{
		Documents: documents,
		Total:     resp.Total,
	})
}

// SaveDocumentDraft godoc
//
//	@Summary		Сохранить черновик документа
//	@Description	Сохраняет черновик документа без создания версии
//	@Tags			content
//	@Security		BearerAuth
//	@Param			document_id	path	string	true	"ID документа"
//	@Param			body	body	models.SaveDraftRequest	true	"Содержание черновика"
//	@Success		200	{object}	models.DocumentResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id}/draft [patch]
func (h *ContentHandler) SaveDocumentDraft(c *gin.Context) {
	documentID := c.Param("document_id")

	var req models.SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.SaveDocumentDraft(ctx, &contentv1.SaveDocumentDraftRequest{
		DocumentId: &commonv1.UUID{Value: documentID},
		Content:    req.Content,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	doc := mapDocument(resp.Document)
	doc.Draft = mapDocumentDraft(resp.Draft)
	c.JSON(http.StatusOK, models.DocumentResponse{
		Document: doc,
	})
}

// CreateDocumentVersion godoc
//
//	@Summary		Создать версию документа
//	@Description	Создаёт новую версию документа и опубликовывает её
//	@Tags			content
//	@Security		BearerAuth
//	@Param			document_id	path	string	true	"ID документа"
//	@Param			body	body	models.CreateVersionRequest	true	"Данные версии"
//	@Success		201	{object}	models.VersionResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id}/versions [post]
func (h *ContentHandler) CreateDocumentVersion(c *gin.Context) {
	documentID := c.Param("document_id")

	var req models.CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.CreateDocumentVersion(ctx, &contentv1.CreateDocumentVersionRequest{
		DocumentId:    &commonv1.UUID{Value: documentID},
		Content:       req.Content,
		ChangeSummary: req.ChangeSummary,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.VersionResponse{
		Version: *mapDocumentVersion(resp.Version),
	})
}

// GetDocumentVersion godoc
//
//	@Summary		Получить версию документа
//	@Description	Получает информацию о конкретной версии документа
//	@Tags			content
//	@Param			version_id	path	string	true	"ID версии"
//	@Success		200	{object}	models.VersionResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/document-versions/{version_id} [get]
func (h *ContentHandler) GetDocumentVersion(c *gin.Context) {
	versionID := c.Param("version_id")

	ctx := forwardAuth(c)
	resp, err := h.client.Client.GetDocumentVersionById(ctx, &contentv1.GetDocumentVersionByIdRequest{
		VersionId: &commonv1.UUID{Value: versionID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.VersionResponse{
		Version: *mapDocumentVersion(resp.Version),
	})
}

// ListDocumentVersions godoc
//
//	@Summary		Список версий документа
//	@Description	Получает список всех версий документа с пагинацией
//	@Tags			content
//	@Param			document_id	path	string	true	"ID документа"
//	@Param			limit	query	int	false	"Лимит записей"	default(10)
//	@Param			offset	query	int	false	"Смещение"	default(0)
//	@Success		200	{object}	models.VersionListResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id}/versions [get]
func (h *ContentHandler) ListDocumentVersions(c *gin.Context) {
	documentID := c.Param("document_id")
	limit := uint32(10)
	offset := uint32(0)

	if l := c.Query("limit"); l != "" {
		var val int
		if _, err := c.GetPostForm(l); err {
			val = 10
		}
		limit = uint32(val)
	}
	if o := c.Query("offset"); o != "" {
		var val int
		offset = uint32(val)
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.ListDocumentVersions(ctx, &contentv1.ListDocumentVersionsRequest{
		DocumentId: &commonv1.UUID{Value: documentID},
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	versions := make([]models.DocumentVersionDetail, len(resp.Versions))
	for i, v := range resp.Versions {
		versions[i] = *mapDocumentVersion(v)
	}

	c.JSON(http.StatusOK, models.VersionListResponse{
		Versions: versions,
		Total:    resp.Total,
	})
}

// RestoreDocumentVersion godoc
//
//	@Summary		Восстановить версию документа
//	@Description	Восстанавливает документ до выбранной версии
//	@Tags			content
//	@Security		BearerAuth
//	@Param			document_id	path	string	true	"ID документа"
//	@Param			version_id	path	string	true	"ID версии для восстановления"
//	@Param			body	body	models.RestoreVersionRequest	true	"Данные восстановления"
//	@Success		200	{object}	models.VersionResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id}/versions/{version_id}/restore [post]
func (h *ContentHandler) RestoreDocumentVersion(c *gin.Context) {
	documentID := c.Param("document_id")
	versionID := c.Param("version_id")

	var req models.RestoreVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.RestoreDocumentVersion(ctx, &contentv1.RestoreDocumentVersionRequest{
		DocumentId:    &commonv1.UUID{Value: documentID},
		VersionId:     &commonv1.UUID{Value: versionID},
		ChangeSummary: req.ChangeSummary,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.VersionResponse{
		Version: *mapDocumentVersion(resp.Version),
	})
}

// DeleteDocument godoc
//
//	@Summary		Удалить документ
//	@Description	Удаляет документ из репозитория
//	@Tags			content
//	@Security		BearerAuth
//	@Param			document_id	path	string	true	"ID документа"
//	@Success		204
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/documents/{document_id} [delete]
func (h *ContentHandler) DeleteDocument(c *gin.Context) {
	documentID := c.Param("document_id")

	ctx := forwardAuth(c)
	_, err := h.client.Client.DeleteDocument(ctx, &contentv1.DeleteDocumentRequest{
		DocumentId: &commonv1.UUID{Value: documentID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateFile godoc
//
//	@Summary		Создать файл
//	@Description	Создаёт новый файл в репозитории
//	@Tags			content
//	@Security		BearerAuth
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Param			body	body	models.CreateFileRequest	true	"Данные файла"
//	@Success		201	{object}	models.FileResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/repositories/{repo_id}/files [post]
func (h *ContentHandler) CreateFile(c *gin.Context) {
	repoID := c.Param("repo_id")

	var req models.CreateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	protoReq := &contentv1.CreateFileRequest{
		RepoId:         &commonv1.UUID{Value: repoID},
		FileName:       req.FileName,
		StorageKey:     req.StorageKey,
		MimeType:       req.MimeType,
		SizeBytes:      req.SizeBytes,
		ChecksumSha256: req.ChecksumSHA256,
		ChangeSummary:  req.ChangeSummary,
	}

	resp, err := h.client.Client.CreateFile(ctx, protoReq)
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.FileResponse{
		File: mapFile(resp.File),
	})
}

// GetFile godoc
//
//	@Summary		Получить файл
//	@Description	Получает информацию о файле по ID
//	@Tags			content
//	@Param			file_id	path	string	true	"ID файла"
//	@Success		200	{object}	models.FileResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/files/{file_id} [get]
func (h *ContentHandler) GetFile(c *gin.Context) {
	fileID := c.Param("file_id")

	ctx := forwardAuth(c)
	resp, err := h.client.Client.GetFileById(ctx, &contentv1.GetFileByIdRequest{
		FileId: &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.FileResponse{
		File: mapFile(resp.File),
	})
}

// ListRepositoryFiles godoc
//
//	@Summary		Список файлов репозитория
//	@Description	Получает список файлов в репозитории с пагинацией
//	@Tags			content
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Param			limit	query	int	false	"Лимит записей"	default(10)
//	@Param			offset	query	int	false	"Смещение"	default(0)
//	@Success		200	{object}	models.FileListResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/repositories/{repo_id}/files [get]
func (h *ContentHandler) ListRepositoryFiles(c *gin.Context) {
	repoID := c.Param("repo_id")
	limit := uint32(10)
	offset := uint32(0)

	if l := c.Query("limit"); l != "" {
		var val int
		limit = uint32(val)
	}
	if o := c.Query("offset"); o != "" {
		var val int
		offset = uint32(val)
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.ListRepositoryFiles(ctx, &contentv1.ListRepositoryFilesRequest{
		RepoId: &commonv1.UUID{Value: repoID},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	files := make([]models.FileDetailResponse, len(resp.Files))
	for i, f := range resp.Files {
		files[i] = mapFile(f)
	}

	c.JSON(http.StatusOK, models.FileListResponse{
		Files: files,
		Total: resp.Total,
	})
}

// AddFileVersion godoc
//
//	@Summary		Добавить версию файла
//	@Description	Добавляет новую версию существующего файла
//	@Tags			content
//	@Security		BearerAuth
//	@Param			file_id	path	string	true	"ID файла"
//	@Param			body	body	models.AddFileVersionRequest	true	"Данные новой версии"
//	@Success		201	{object}	models.FileVersionResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/files/{file_id}/versions [post]
func (h *ContentHandler) AddFileVersion(c *gin.Context) {
	fileID := c.Param("file_id")

	var req models.AddFileVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.AddFileVersion(ctx, &contentv1.AddFileVersionRequest{
		FileId:         &commonv1.UUID{Value: fileID},
		StorageKey:     req.StorageKey,
		MimeType:       req.MimeType,
		SizeBytes:      req.SizeBytes,
		ChecksumSha256: req.ChecksumSHA256,
		ChangeSummary:  req.ChangeSummary,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.FileVersionResponse{
		Version: mapFileVersion(resp.Version),
	})
}

// GetFileVersion godoc
//
//	@Summary		Получить версию файла
//	@Description	Получает информацию о конкретной версии файла
//	@Tags			content
//	@Param			version_id	path	string	true	"ID версии файла"
//	@Success		200	{object}	models.FileVersionResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/file-versions/{version_id} [get]
func (h *ContentHandler) GetFileVersion(c *gin.Context) {
	versionID := c.Param("version_id")

	ctx := forwardAuth(c)
	resp, err := h.client.Client.GetFileVersionById(ctx, &contentv1.GetFileVersionByIdRequest{
		VersionId: &commonv1.UUID{Value: versionID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.FileVersionResponse{
		Version: mapFileVersion(resp.Version),
	})
}

// ListFileVersions godoc
//
//	@Summary		Список версий файла
//	@Description	Получает список всех версий файла с пагинацией
//	@Tags			content
//	@Param			file_id	path	string	true	"ID файла"
//	@Param			limit	query	int	false	"Лимит записей"	default(10)
//	@Param			offset	query	int	false	"Смещение"	default(0)
//	@Success		200	{object}	models.FileVersionListResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/files/{file_id}/versions [get]
func (h *ContentHandler) ListFileVersions(c *gin.Context) {
	fileID := c.Param("file_id")
	limit := uint32(10)
	offset := uint32(0)

	if l := c.Query("limit"); l != "" {
		var val int
		limit = uint32(val)
	}
	if o := c.Query("offset"); o != "" {
		var val int
		offset = uint32(val)
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.ListFileVersions(ctx, &contentv1.ListFileVersionsRequest{
		FileId: &commonv1.UUID{Value: fileID},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	versions := make([]models.FileVersionDetail, len(resp.Versions))
	for i, v := range resp.Versions {
		versions[i] = mapFileVersion(v)
	}

	c.JSON(http.StatusOK, models.FileVersionListResponse{
		Versions: versions,
		Total:    resp.Total,
	})
}

// RestoreFileVersion godoc
//
//	@Summary		Восстановить версию файла
//	@Description	Восстанавливает файл до выбранной версии
//	@Tags			content
//	@Security		BearerAuth
//	@Param			file_id	path	string	true	"ID файла"
//	@Param			version_id	path	string	true	"ID версии для восстановления"
//	@Param			body	body	models.RestoreFileVersionRequest	true	"Данные восстановления"
//	@Success		200	{object}	models.FileVersionResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/files/{file_id}/versions/{version_id}/restore [post]
func (h *ContentHandler) RestoreFileVersion(c *gin.Context) {
	fileID := c.Param("file_id")
	versionID := c.Param("version_id")

	var req models.RestoreFileVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.RestoreFileVersion(ctx, &contentv1.RestoreFileVersionRequest{
		FileId:        &commonv1.UUID{Value: fileID},
		VersionId:     &commonv1.UUID{Value: versionID},
		ChangeSummary: req.ChangeSummary,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.FileVersionResponse{
		Version: mapFileVersion(resp.Version),
	})
}

// DeleteFile godoc
//
//	@Summary		Удалить файл
//	@Description	Удаляет файл из репозитория
//	@Tags			content
//	@Security		BearerAuth
//	@Param			file_id	path	string	true	"ID файла"
//	@Success		204
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		404	{object}	models.ErrorResponse
//	@Router			/files/{file_id} [delete]
func (h *ContentHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("file_id")

	ctx := forwardAuth(c)
	_, err := h.client.Client.DeleteFile(ctx, &contentv1.DeleteFileRequest{
		FileId: &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}
