package handlers

import (
	"net/http"
	"sort"
	"strings"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SearchHandler struct {
	client        *clients.SearchClient
	profileClient *clients.ProfileClient
}

func NewSearchHandler(client *clients.SearchClient, profileClient *clients.ProfileClient) *SearchHandler {
	return &SearchHandler{client: client, profileClient: profileClient}
}

// Search godoc
//
//	@Summary		Поиск по индексам
//	@Description	Выполняет полнотекстовый поиск по репозиториям, документам и файлам
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.SearchRequest	true	"Параметры поиска"
//	@Success		200		{object}	models.SearchResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/search [post]
func (h *SearchHandler) Search(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	entityTypes, err := toProtoSearchEntityTypes(req.EntityTypes)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	tagID, err := optionalUUIDFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid tag_id format",
		})
		return
	}

	ownerID, err := optionalUUIDFromString(req.OwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid owner_id format",
		})
		return
	}

	repoID, err := optionalUUIDFromString(req.RepoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	ctx := forwardAuth(c)
	resp, err := h.client.Client.Search(ctx, &searchv1.SearchRequest{
		Query:       req.Query,
		EntityTypes: entityTypes,
		TagId:       tagID,
		OwnerId:     ownerID,
		RepoId:      repoID,
		Limit:       req.Limit,
		Offset:      req.Offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	hits := make([]models.SearchHitResponse, len(resp.GetHits()))
	for i, hit := range resp.GetHits() {
		hits[i] = mapSearchHit(hit)
	}

	c.JSON(http.StatusOK, models.SearchResponse{
		Hits:  hits,
		Total: resp.GetTotal(),
	})
}

// SearchUsers godoc
//
//	@Summary		Поиск пользователей
//	@Description	Ищет пользователей по профилям и репозиториям с опциональными фильтрами по вузу и предмету
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.SearchUsersRequest	true	"Параметры поиска пользователей"
//	@Success		200		{object}	models.SearchUsersResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/search/users [post]
func (h *SearchHandler) SearchUsers(c *gin.Context) {
	var req models.SearchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		c.JSON(http.StatusOK, models.SearchUsersResponse{Users: []models.SearchUserHitResponse{}, Total: 0})
		return
	}

	tagID, err := optionalUUIDFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid tag_id format"})
		return
	}

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 20
	}

	offset := req.Offset
	requestedCount := int(limit + offset)
	if requestedCount < 20 {
		requestedCount = 20
	}
	if requestedCount > 100 {
		requestedCount = 100
	}

	ctx := forwardAuth(c)
	searchResp, err := h.client.Client.Search(ctx, &searchv1.SearchRequest{
		Query:       query,
		EntityTypes: []commonv1.SearchEntityType{commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_REPOSITORY},
		TagId:       tagID,
		Limit:       uint32(requestedCount),
		Offset:      0,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	repositoriesCountByOwner := make(map[string]uint32)
	for _, hit := range searchResp.GetHits() {
		owner := strings.TrimSpace(hit.GetOwnerId().GetValue())
		if owner == "" {
			continue
		}
		repositoriesCountByOwner[owner] += 1
	}

	queryLower := normalizeSearchQuery(query)
	requestedUniversity := normalizeUniversity(req.University)

	users := make([]models.SearchUserHitResponse, 0, len(repositoriesCountByOwner))
	for ownerID, reposCount := range repositoriesCountByOwner {
		profileResp, profileErr := h.profileClient.Client.GetProfileByUserId(ctx, &profilev1.GetProfileByUserIdRequest{
			UserId: &commonv1.UUID{Value: ownerID},
		})
		if profileErr != nil {
			continue
		}

		profile := profileResp.GetProfile()
		username := strings.TrimSpace(profile.GetUsername())
		displayName := strings.TrimSpace(profile.GetDisplayName())

		if !matchesUserQuery(queryLower, username, displayName) {
			continue
		}

		university := normalizeUniversity(profile.GetLocation())
		if requestedUniversity != "" && university != requestedUniversity {
			continue
		}

		users = append(users, models.SearchUserHitResponse{
			UserID:            profile.GetUserId().GetValue(),
			Username:          username,
			DisplayName:       displayName,
			AvatarURL:         profile.GetAvatarUrl(),
			University:        firstNonEmpty(university, strings.TrimSpace(profile.GetLocation())),
			RepositoriesCount: reposCount,
		})
	}

	sort.SliceStable(users, func(i, j int) bool {
		if users[i].RepositoriesCount != users[j].RepositoriesCount {
			return users[i].RepositoriesCount > users[j].RepositoriesCount
		}
		if users[i].Username != users[j].Username {
			return users[i].Username < users[j].Username
		}
		return users[i].UserID < users[j].UserID
	})

	total := uint64(len(users))
	if int(offset) >= len(users) {
		c.JSON(http.StatusOK, models.SearchUsersResponse{Users: []models.SearchUserHitResponse{}, Total: total})
		return
	}

	start := int(offset)
	end := int(offset + limit)
	if end > len(users) {
		end = len(users)
	}

	c.JSON(http.StatusOK, models.SearchUsersResponse{Users: users[start:end], Total: total})
}

// UpsertRepositoryIndex godoc
//
//	@Summary		Обновить индекс репозитория
//	@Description	Добавляет или обновляет индексную запись репозитория
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			request	body	models.UpsertRepositoryIndexRequest	true	"Данные индекса"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/repositories [post]
func (h *SearchHandler) UpsertRepositoryIndex(c *gin.Context) {
	var req models.UpsertRepositoryIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	repoID, err := uuidFromString(req.RepoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid repo_id format"})
		return
	}
	ownerID, err := uuidFromString(req.OwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid owner_id format"})
		return
	}
	tagID, err := uuidFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid tag_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.UpsertRepositoryIndex(ctx, &searchv1.UpsertRepositoryIndexRequest{
		RepoId:      repoID,
		OwnerId:     ownerID,
		TagId:       tagID,
		Title:       req.Title,
		Description: req.Description,
		TagName:     req.TagName,
		IsPublic:    req.IsPublic,
		UpdatedAt:   timestamppb.New(req.UpdatedAt),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteRepositoryIndex godoc
//
//	@Summary		Удалить индекс репозитория
//	@Description	Удаляет индексную запись репозитория
//	@Tags			search
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверный формат ID"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/repositories/{repo_id} [delete]
func (h *SearchHandler) DeleteRepositoryIndex(c *gin.Context) {
	repoID, err := uuidFromString(c.Param("repo_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid repo_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.DeleteRepositoryIndex(ctx, &searchv1.DeleteRepositoryIndexRequest{RepoId: repoID})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpsertDocumentIndex godoc
//
//	@Summary		Обновить индекс документа
//	@Description	Добавляет или обновляет индексную запись документа
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			request	body	models.UpsertDocumentIndexRequest	true	"Данные индекса"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/documents [post]
func (h *SearchHandler) UpsertDocumentIndex(c *gin.Context) {
	var req models.UpsertDocumentIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	documentID, err := uuidFromString(req.DocumentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid document_id format"})
		return
	}
	repoID, err := uuidFromString(req.RepoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid repo_id format"})
		return
	}
	ownerID, err := uuidFromString(req.OwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid owner_id format"})
		return
	}
	tagID, err := uuidFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid tag_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.UpsertDocumentIndex(ctx, &searchv1.UpsertDocumentIndexRequest{
		DocumentId: documentID,
		RepoId:     repoID,
		OwnerId:    ownerID,
		TagId:      tagID,
		Title:      req.Title,
		Content:    req.Content,
		TagName:    req.TagName,
		IsPublic:   req.IsPublic,
		UpdatedAt:  timestamppb.New(req.UpdatedAt),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteDocumentIndex godoc
//
//	@Summary		Удалить индекс документа
//	@Description	Удаляет индексную запись документа
//	@Tags			search
//	@Param			document_id	path	string	true	"ID документа"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверный формат ID"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/documents/{document_id} [delete]
func (h *SearchHandler) DeleteDocumentIndex(c *gin.Context) {
	documentID, err := uuidFromString(c.Param("document_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid document_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.DeleteDocumentIndex(ctx, &searchv1.DeleteDocumentIndexRequest{DocumentId: documentID})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpsertFileIndex godoc
//
//	@Summary		Обновить индекс файла
//	@Description	Добавляет или обновляет индексную запись файла
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			request	body	models.UpsertFileIndexRequest	true	"Данные индекса"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/files [post]
func (h *SearchHandler) UpsertFileIndex(c *gin.Context) {
	var req models.UpsertFileIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	fileID, err := uuidFromString(req.FileID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid file_id format"})
		return
	}
	repoID, err := uuidFromString(req.RepoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid repo_id format"})
		return
	}
	ownerID, err := uuidFromString(req.OwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid owner_id format"})
		return
	}
	tagID, err := uuidFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid tag_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.UpsertFileIndex(ctx, &searchv1.UpsertFileIndexRequest{
		FileId:    fileID,
		RepoId:    repoID,
		OwnerId:   ownerID,
		TagId:     tagID,
		FileName:  req.FileName,
		MimeType:  req.MimeType,
		TagName:   req.TagName,
		IsPublic:  req.IsPublic,
		UpdatedAt: timestamppb.New(req.UpdatedAt),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteFileIndex godoc
//
//	@Summary		Удалить индекс файла
//	@Description	Удаляет индексную запись файла
//	@Tags			search
//	@Param			file_id	path	string	true	"ID файла"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверный формат ID"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/search/index/files/{file_id} [delete]
func (h *SearchHandler) DeleteFileIndex(c *gin.Context) {
	fileID, err := uuidFromString(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "invalid file_id format"})
		return
	}

	ctx := forwardAuth(c)
	_, err = h.client.Client.DeleteFileIndex(ctx, &searchv1.DeleteFileIndexRequest{FileId: fileID})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

func optionalUUIDFromString(s string) (*commonv1.UUID, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, nil
	}
	return uuidFromString(trimmed)
}

func toProtoSearchEntityTypes(values []string) ([]commonv1.SearchEntityType, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]commonv1.SearchEntityType, 0, len(values))
	for _, raw := range values {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "repository":
			result = append(result, commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_REPOSITORY)
		case "document":
			result = append(result, commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_DOCUMENT)
		case "file":
			result = append(result, commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_FILE)
		case "", "unspecified":
			continue
		default:
			return nil, errInvalidEntityType(raw)
		}
	}

	return result, nil
}

func normalizeSearchQuery(value string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.ToLower(value), "@"))
}

func normalizeUniversity(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	if strings.Contains(normalized, "мирэа") || strings.Contains(normalized, "mirea") {
		return "МИРЭА"
	}
	if strings.Contains(normalized, "мгу") || strings.Contains(normalized, "msu") {
		return "МГУ"
	}
	return ""
}

func matchesUserQuery(queryLower, username, displayName string) bool {
	if queryLower == "" {
		return true
	}
	combined := strings.ToLower(strings.TrimSpace(username + " " + displayName))
	return strings.Contains(combined, queryLower)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type errInvalidEntityType string

func (e errInvalidEntityType) Error() string {
	return "invalid entity_type: " + string(e)
}
