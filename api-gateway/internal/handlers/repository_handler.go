package handlers

import (
	"net/http"
	"strconv"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
)

type RepositoryHandler struct {
	client *clients.RepositoryClient
}

func NewRepositoryHandler(client *clients.RepositoryClient) *RepositoryHandler {
	return &RepositoryHandler{client: client}
}

// CreateRepository godoc
//
//	@Summary		Создание нового репозитория
//	@Description	Создаёт новый репозиторий для текущего пользователя
//	@Tags			repositories
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CreateRepositoryRequest	true	"Данные репозитория"
//	@Success		201		{object}	models.CreateRepositoryResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		404		{object}	models.ErrorResponse	"Тег не найден"
//	@Failure		409		{object}	models.ErrorResponse	"Slug уже занят"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories [post]
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req models.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)

	tagID, err := uuidFromString(req.TagID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid tag_id format",
		})
		return
	}

	resp, err := h.client.Client.CreateRepository(ctx, &repositoryv1.CreateRepositoryRequest{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		TagId:       tagID,
		Visibility:  toProtoVisibility(req.Visibility),
		Type:        toProtoType(req.Type),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.CreateRepositoryResponse{
		Repository: mapRepository(resp.GetRepository()),
	})
}

// GetRepositoryByID godoc
//
//	@Summary		Получение репозитория по ID
//	@Description	Возвращает информацию о репозитории по его ID
//	@Tags			repositories
//	@Produce		json
//	@Param			repo_id	path		string	true	"ID репозитория"
//	@Success		200		{object}	models.GetRepositoryResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверный формат ID"
//	@Failure		403		{object}	models.ErrorResponse	"Доступ запрещён"
//	@Failure		404		{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/{repo_id} [get]
func (h *RepositoryHandler) GetRepositoryByID(c *gin.Context) {
	repoID := c.Param("repo_id")

	uuid, err := uuidFromString(repoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetRepositoryById(ctx, &repositoryv1.GetRepositoryByIdRequest{
		RepoId: uuid,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetRepositoryResponse{
		Repository: mapRepository(resp.GetRepository()),
	})
}

// GetRepositoryBySlug godoc
//
//	@Summary		Получение репозитория по slug
//	@Description	Возвращает информацию о репозитории по owner nickname (или owner_id) и slug
//	@Tags			repositories
//	@Produce		json
//	@Param			owner_id	path		string	true	"Nickname владельца или owner_id"
//	@Param			slug		path		string	true	"Slug репозитория"
//	@Success		200			{object}	models.GetRepositoryResponse
//	@Failure		400			{object}	models.ErrorResponse	"Неверный формат данных"
//	@Failure		403			{object}	models.ErrorResponse	"Доступ запрещён"
//	@Failure		404			{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		500			{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{owner_id}/repositories/{slug} [get]
func (h *RepositoryHandler) GetRepositoryBySlug(c *gin.Context) {
	ownerKey := c.Param("owner_id")
	slug := c.Param("slug")

	if ownerKey == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "owner is required",
		})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetRepositoryBySlug(ctx, &repositoryv1.GetRepositoryBySlugRequest{
		OwnerId: &commonv1.UUID{Value: ownerKey},
		Slug:    slug,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetRepositoryResponse{
		Repository: mapRepository(resp.GetRepository()),
	})
}

// UpdateRepository godoc
//
//	@Summary		Обновление репозитория
//	@Description	Обновляет информацию о репозитории (только владелец может обновлять)
//	@Tags			repositories
//	@Accept			json
//	@Produce		json
//	@Param			repo_id	path		string							true	"ID репозитория"
//	@Param			request	body		models.UpdateRepositoryRequest	true	"Данные для обновления"
//	@Success		200		{object}	models.UpdateRepositoryResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	models.ErrorResponse	"Доступ запрещён"
//	@Failure		404		{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		409		{object}	models.ErrorResponse	"Slug уже занят"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/{repo_id} [patch]
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	repoID := c.Param("repo_id")

	uuid, err := uuidFromString(repoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	var req models.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)

	var tagID *commonv1.UUID
	if req.TagID != "" {
		parsed, err := uuidFromString(req.TagID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "invalid tag_id format",
			})
			return
		}
		tagID = parsed
	}

	resp, err := h.client.Client.UpdateRepository(ctx, &repositoryv1.UpdateRepositoryRequest{
		RepoId:      uuid,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		TagId:       tagID,
		Visibility:  toProtoVisibility(req.Visibility),
		Type:        toProtoType(req.Type),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.UpdateRepositoryResponse{
		Repository: mapRepository(resp.GetRepository()),
	})
}

// DeleteRepository godoc
//
//	@Summary		Удаление репозитория
//	@Description	Удаляет репозиторий (только владелец может удалять)
//	@Tags			repositories
//	@Param			repo_id	path	string	true	"ID репозитория"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse	"Неверный формат ID"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	models.ErrorResponse	"Доступ запрещён"
//	@Failure		404		{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/{repo_id} [delete]
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	repoID := c.Param("repo_id")

	uuid, err := uuidFromString(repoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	ctx := forwardAuth(c)

	_, err = h.client.Client.DeleteRepository(ctx, &repositoryv1.DeleteRepositoryRequest{
		RepoId: uuid,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// ForkRepository godoc
//
//	@Summary		Форк репозитория
//	@Description	Создаёт копию публичного репозитория от другого пользователя
//	@Tags			repositories
//	@Accept			json
//	@Produce		json
//	@Param			repo_id	path		string						true	"ID оригинального репозитория"
//	@Param			request	body		models.ForkRepositoryRequest	true	"Данные для форка"
//	@Success		201		{object}	models.ForkRepositoryResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные данные"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	models.ErrorResponse	"Репозиторий приватный или это твой репозиторий"
//	@Failure		404		{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		409		{object}	models.ErrorResponse	"Slug уже занят"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/{repo_id}/fork [post]
func (h *RepositoryHandler) ForkRepository(c *gin.Context) {
	repoID := c.Param("repo_id")

	uuid, err := uuidFromString(repoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	var req models.ForkRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.ForkRepository(ctx, &repositoryv1.ForkRepositoryRequest{
		SourceRepoId: uuid,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, models.ForkRepositoryResponse{
		Repository: mapRepository(resp.GetRepository()),
	})
}

// ListMyRepositories godoc
//
//	@Summary		Получение моих репозиториев
//	@Description	Возвращает список всех репозиториев текущего пользователя
//	@Tags			repositories
//	@Produce		json
//	@Param			limit	query		integer	false	"Лимит элементов"	default(10)
//	@Param			offset	query		integer	false	"Смещение"			default(0)
//	@Success		200		{object}	models.ListRepositoriesResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные параметры"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/me [get]
func (h *RepositoryHandler) ListMyRepositories(c *gin.Context) {
	limit, offset := parsePagination(c)

	ctx := forwardAuth(c)

	resp, err := h.client.Client.ListMyRepositories(ctx, &repositoryv1.ListMyRepositoriesRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	repos := make([]models.RepositoryResponse, len(resp.GetRepositories()))
	for i, r := range resp.GetRepositories() {
		repos[i] = mapRepository(r)
	}

	c.JSON(http.StatusOK, models.ListRepositoriesResponse{
		Repositories: repos,
		Total:        resp.GetTotal(),
	})
}

// ListUserRepositories godoc
//
//	@Summary		Получение репозиториев пользователя
//	@Description	Возвращает список публичных репозиториев указанного пользователя
//	@Tags			repositories
//	@Produce		json
//	@Param			owner_id	path		string	true	"Nickname владельца или owner_id"
//	@Param			limit		query		integer	false	"Лимит элементов"	default(10)
//	@Param			offset		query		integer	false	"Смещение"			default(0)
//	@Success		200			{object}	models.ListRepositoriesResponse
//	@Failure		400			{object}	models.ErrorResponse	"Неверные параметры"
//	@Failure		401			{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		404			{object}	models.ErrorResponse	"Пользователь не найден"
//	@Failure		500			{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{owner_id}/repositories [get]
func (h *RepositoryHandler) ListUserRepositories(c *gin.Context) {
	ownerKey := c.Param("owner_id")
	if ownerKey == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "owner is required",
		})
		return
	}

	limit, offset := parsePagination(c)

	ctx := forwardAuth(c)

	resp, err := h.client.Client.ListUserRepositories(ctx, &repositoryv1.ListUserRepositoriesRequest{
		OwnerId: &commonv1.UUID{Value: ownerKey},
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	repos := make([]models.RepositoryResponse, len(resp.GetRepositories()))
	for i, r := range resp.GetRepositories() {
		repos[i] = mapRepository(r)
	}

	c.JSON(http.StatusOK, models.ListRepositoriesResponse{
		Repositories: repos,
		Total:        resp.GetTotal(),
	})
}

// ListForks godoc
//
//	@Summary		Получение форков репозитория
//	@Description	Возвращает список всех форков указанного репозитория
//	@Tags			repositories
//	@Produce		json
//	@Param			repo_id	path		string	true	"ID оригинального репозитория"
//	@Param			limit	query		integer	false	"Лимит элементов"	default(10)
//	@Param			offset	query		integer	false	"Смещение"			default(0)
//	@Success		200		{object}	models.ListRepositoriesResponse
//	@Failure		400		{object}	models.ErrorResponse	"Неверные параметры"
//	@Failure		401		{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	models.ErrorResponse	"Доступ запрещён"
//	@Failure		404		{object}	models.ErrorResponse	"Репозиторий не найден"
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/{repo_id}/forks [get]
func (h *RepositoryHandler) ListForks(c *gin.Context) {
	repoID := c.Param("repo_id")

	uuid, err := uuidFromString(repoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid repo_id format",
		})
		return
	}

	limit, offset := parsePagination(c)

	ctx := forwardAuth(c)

	resp, err := h.client.Client.ListForks(ctx, &repositoryv1.ListForksRequest{
		RepoId: uuid,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	repos := make([]models.RepositoryResponse, len(resp.GetRepositories()))
	for i, r := range resp.GetRepositories() {
		repos[i] = mapRepository(r)
	}

	c.JSON(http.StatusOK, models.ListRepositoriesResponse{
		Repositories: repos,
		Total:        resp.GetTotal(),
	})
}

// ListRepositoryTags godoc
//
//	@Summary		Получение всех тегов репозиториев
//	@Description	Возвращает список всех доступных тегов для категоризации репозиториев
//	@Tags			repositories
//	@Produce		json
//	@Success		200	{object}	models.ListRepositoryTagsResponse
//	@Failure		401	{object}	models.ErrorResponse	"Не авторизован"
//	@Failure		500	{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/repositories/tags [get]
func (h *RepositoryHandler) ListRepositoryTags(c *gin.Context) {
	ctx := forwardAuth(c)

	resp, err := h.client.Client.ListRepositoryTags(ctx, &emptypb.Empty{})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	tags := make([]models.RepositoryTagResponse, len(resp.GetTags()))
	for i, t := range resp.GetTags() {
		tags[i] = mapRepositoryTag(t)
	}

	c.JSON(http.StatusOK, models.ListRepositoryTagsResponse{
		Tags: tags,
	})
}

// Helper functions

// parsePagination extracts and validates limit and offset from query parameters.
func parsePagination(c *gin.Context) (uint32, uint32) {
	limit := uint32(10)
	offset := uint32(0)

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseUint(l, 10, 32); err == nil {
			if parsed >= 1 && parsed <= 100 {
				limit = uint32(parsed)
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.ParseUint(o, 10, 32); err == nil {
			offset = uint32(parsed)
		}
	}

	return limit, offset
}

// uuidFromString converts string to *commonv1.UUID.
func uuidFromString(s string) (*commonv1.UUID, error) {
	return &commonv1.UUID{Value: s}, nil
}

// toProtoVisibility converts string visibility to proto enum.
func toProtoVisibility(v models.RepositoryVisibility) commonv1.RepositoryVisibility {
	switch v {
	case models.RepositoryVisibilityPrivate:
		return commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PRIVATE
	default:
		return commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PUBLIC
	}
}

// toProtoType converts string type to proto enum.
func toProtoType(t models.RepositoryType) commonv1.RepositoryType {
	switch t {
	case models.RepositoryTypeNotes:
		return commonv1.RepositoryType_REPOSITORY_TYPE_NOTES
	case models.RepositoryTypeMixed:
		return commonv1.RepositoryType_REPOSITORY_TYPE_MIXED
	default:
		return commonv1.RepositoryType_REPOSITORY_TYPE_ARTICLE
	}
}
