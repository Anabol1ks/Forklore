package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProfileHandler struct {
	client *clients.ProfileClient
}

func NewProfileHandler(client *clients.ProfileClient) *ProfileHandler {
	return &ProfileHandler{client: client}
}

// GetMyProfile godoc
//
//	@Summary		Получить мой профиль
//	@Description	Возвращает профиль текущего авторизованного пользователя
//	@Tags			profiles
//	@Produce		json
//	@Success		200	{object}	models.GetProfileResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetMyProfile(ctx, &emptypb.Empty{})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

// GetProfileByUserID godoc
//
//	@Summary		Получить профиль по user_id
//	@Description	Возвращает профиль пользователя по user_id
//	@Tags			profiles
//	@Produce		json
//	@Param			user_id	path		string	true	"ID пользователя"
//	@Success		200		{object}	models.GetProfileResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/profiles/by-user/{user_id} [get]
func (h *ProfileHandler) GetProfileByUserID(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "user_id is required"})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetProfileByUserId(ctx, &profilev1.GetProfileByUserIdRequest{
		UserId: &commonv1.UUID{Value: userID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

// GetProfileByUsername godoc
//
//	@Summary		Получить профиль по username
//	@Description	Возвращает профиль пользователя по username
//	@Tags			profiles
//	@Produce		json
//	@Param			username	path		string	true	"Username пользователя"
//	@Success		200			{object}	models.GetProfileResponse
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		404			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Router			/profiles/by-username/{username} [get]
func (h *ProfileHandler) GetProfileByUsername(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "username is required"})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetProfileByUsername(ctx, &profilev1.GetProfileByUsernameRequest{Username: username})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

// UpdateProfile godoc
//
//	@Summary		Обновить профиль
//	@Description	Обновляет публичные поля профиля текущего пользователя
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.UpdateProfileRequest	true	"Данные профиля"
//	@Success		200		{object}	models.GetProfileResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/me [patch]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.UpdateProfile(ctx, &profilev1.UpdateProfileRequest{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarUrl:   req.AvatarURL,
		CoverUrl:    req.CoverURL,
		Location:    req.Location,
		WebsiteUrl:  req.WebsiteURL,
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

// UpdateProfileReadme godoc
//
//	@Summary		Обновить README профиля
//	@Description	Обновляет readme_markdown текущего пользователя
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.UpdateProfileReadmeRequest	true	"README профиля"
//	@Success		200		{object}	models.GetProfileResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/me/readme [patch]
func (h *ProfileHandler) UpdateProfileReadme(c *gin.Context) {
	var req models.UpdateProfileReadmeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.UpdateProfileReadme(ctx, &profilev1.UpdateProfileReadmeRequest{
		ReadmeMarkdown: req.ReadmeMarkdown,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

// ListProfileSocialLinks godoc
//
//	@Summary		Список соцссылок профиля
//	@Description	Возвращает социальные ссылки профиля по user_id
//	@Tags			profiles
//	@Produce		json
//	@Param			user_id	path		string	true	"ID пользователя"
//	@Success		200		{object}	models.ListProfileSocialLinksResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/profiles/{user_id}/social-links [get]
func (h *ProfileHandler) ListProfileSocialLinks(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "user_id is required"})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.ListProfileSocialLinks(ctx, &profilev1.ListProfileSocialLinksRequest{
		UserId: &commonv1.UUID{Value: userID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	links := make([]models.ProfileSocialLinkResponse, len(resp.GetSocialLinks()))
	for i, link := range resp.GetSocialLinks() {
		links[i] = mapProfileSocialLink(link)
	}

	c.JSON(http.StatusOK, models.ListProfileSocialLinksResponse{SocialLinks: links})
}

// UpsertProfileSocialLink godoc
//
//	@Summary		Создать или обновить соцссылку
//	@Description	Создаёт новую соцссылку или обновляет существующую по social_link_id
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.UpsertProfileSocialLinkRequest	true	"Данные соцссылки"
//	@Success		200		{object}	models.UpsertProfileSocialLinkResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/social-links [post]
//	@Router			/profiles/social-links [put]
func (h *ProfileHandler) UpsertProfileSocialLink(c *gin.Context) {
	var req models.UpsertProfileSocialLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	platform, err := toProtoSocialPlatform(req.Platform)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	ctx := forwardAuth(c)

	socialLinkID := strings.TrimSpace(req.SocialLinkID)
	grpcReq := &profilev1.UpsertProfileSocialLinkRequest{
		Platform:  platform,
		Url:       req.URL,
		Label:     req.Label,
		Position:  req.Position,
		IsVisible: req.IsVisible,
	}
	if socialLinkID != "" {
		grpcReq.SocialLinkId = &commonv1.UUID{Value: socialLinkID}
	}

	resp, err := h.client.Client.UpsertProfileSocialLink(ctx, grpcReq)
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.UpsertProfileSocialLinkResponse{SocialLink: mapProfileSocialLink(resp.GetSocialLink())})
}

// DeleteProfileSocialLink godoc
//
//	@Summary		Удалить соцссылку
//	@Description	Удаляет соцссылку текущего пользователя по social_link_id
//	@Tags			profiles
//	@Produce		json
//	@Param			social_link_id	path		string	true	"ID соцссылки"
//	@Success		204			"No Content"
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		401			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/social-links/{social_link_id} [delete]
func (h *ProfileHandler) DeleteProfileSocialLink(c *gin.Context) {
	socialLinkID := strings.TrimSpace(c.Param("social_link_id"))
	if socialLinkID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "social_link_id is required"})
		return
	}

	ctx := forwardAuth(c)

	_, err := h.client.Client.DeleteProfileSocialLink(ctx, &profilev1.DeleteProfileSocialLinkRequest{
		SocialLinkId: &commonv1.UUID{Value: socialLinkID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// FollowUser godoc
//
//	@Summary		Подписаться на пользователя
//	@Description	Создаёт подписку текущего пользователя на пользователя followee_id
//	@Tags			profiles
//	@Produce		json
//	@Param			followee_id	path	string	true	"ID пользователя, на которого подписываемся"
//	@Success		204			"No Content"
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		401			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/{followee_id}/follow [post]
func (h *ProfileHandler) FollowUser(c *gin.Context) {
	followeeID := strings.TrimSpace(c.Param("followee_id"))
	if followeeID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "followee_id is required"})
		return
	}

	ctx := forwardAuth(c)

	_, err := h.client.Client.FollowUser(ctx, &profilev1.FollowUserRequest{
		FolloweeId: &commonv1.UUID{Value: followeeID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// UnfollowUser godoc
//
//	@Summary		Отписаться от пользователя
//	@Description	Удаляет подписку текущего пользователя на пользователя followee_id
//	@Tags			profiles
//	@Produce		json
//	@Param			followee_id	path	string	true	"ID пользователя, от которого отписываемся"
//	@Success		204			"No Content"
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		401			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/{followee_id}/follow [delete]
func (h *ProfileHandler) UnfollowUser(c *gin.Context) {
	followeeID := strings.TrimSpace(c.Param("followee_id"))
	if followeeID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "followee_id is required"})
		return
	}

	ctx := forwardAuth(c)

	_, err := h.client.Client.UnfollowUser(ctx, &profilev1.UnfollowUserRequest{
		FolloweeId: &commonv1.UUID{Value: followeeID},
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListFollowers godoc
//
//	@Summary		Список подписчиков пользователя
//	@Description	Возвращает подписчиков пользователя user_id
//	@Tags			profiles
//	@Produce		json
//	@Param			user_id	path		string	true	"ID пользователя"
//	@Param			limit	query		int		false	"Лимит (1..100)" default(20)
//	@Param			offset	query		int		false	"Смещение" default(0)
//	@Success		200		{object}	models.ListProfilePreviewsResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/profiles/{user_id}/followers [get]
func (h *ProfileHandler) ListFollowers(c *gin.Context) {
	h.listProfilePreviews(c, true)
}

// ListFollowing godoc
//
//	@Summary		Список подписок пользователя
//	@Description	Возвращает пользователей, на которых подписан user_id
//	@Tags			profiles
//	@Produce		json
//	@Param			user_id	path		string	true	"ID пользователя"
//	@Param			limit	query		int		false	"Лимит (1..100)" default(20)
//	@Param			offset	query		int		false	"Смещение" default(0)
//	@Success		200		{object}	models.ListProfilePreviewsResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/profiles/{user_id}/following [get]
func (h *ProfileHandler) ListFollowing(c *gin.Context) {
	h.listProfilePreviews(c, false)
}

// ListAvailableTitles godoc
//
//	@Summary		Список доступных титулов
//	@Description	Возвращает активные титулы профиля
//	@Tags			profiles
//	@Produce		json
//	@Success		200		{object}	models.ListAvailableTitlesResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/profiles/titles [get]
func (h *ProfileHandler) ListAvailableTitles(c *gin.Context) {
	resp, err := h.client.Client.ListAvailableTitles(c.Request.Context(), &emptypb.Empty{})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	titles := make([]models.ProfileTitleResponse, len(resp.GetTitles()))
	for i, title := range resp.GetTitles() {
		titles[i] = mapProfileTitle(title)
	}

	c.JSON(http.StatusOK, models.ListAvailableTitlesResponse{Titles: titles})
}

// SetProfileTitle godoc
//
//	@Summary		Установить титул профиля
//	@Description	Устанавливает title_code для текущего пользователя
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.SetProfileTitleRequest	true	"Титул профиля"
//	@Success		200		{object}	models.GetProfileResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/profiles/me/title [put]
func (h *ProfileHandler) SetProfileTitle(c *gin.Context) {
	var req models.SetProfileTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	ctx := forwardAuth(c)

	resp, err := h.client.Client.SetProfileTitle(ctx, &profilev1.SetProfileTitleRequest{TitleCode: req.TitleCode})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetProfileResponse{Profile: mapProfile(resp.GetProfile())})
}

func (h *ProfileHandler) listProfilePreviews(c *gin.Context, followers bool) {
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "user_id is required"})
		return
	}

	limit := uint32(20)
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || parsed == 0 || parsed > 100 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "limit must be between 1 and 100"})
			return
		}
		limit = uint32(parsed)
	}

	offset := uint32(0)
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "offset must be a non-negative integer"})
			return
		}
		offset = uint32(parsed)
	}

	ctx := forwardAuth(c)

	var (
		resp *profilev1.ListProfilePreviewsResponse
		err  error
	)
	if followers {
		resp, err = h.client.Client.ListFollowers(ctx, &profilev1.ListFollowersRequest{
			UserId: &commonv1.UUID{Value: userID},
			Limit:  limit,
			Offset: offset,
		})
	} else {
		resp, err = h.client.Client.ListFollowing(ctx, &profilev1.ListFollowingRequest{
			UserId: &commonv1.UUID{Value: userID},
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	items := make([]models.ProfilePreviewResponse, len(resp.GetProfiles()))
	for i, p := range resp.GetProfiles() {
		items[i] = mapProfilePreview(p)
	}

	c.JSON(http.StatusOK, models.ListProfilePreviewsResponse{
		Profiles: items,
		Total:    resp.GetTotal(),
	})
}

func toProtoSocialPlatform(value string) (profilev1.SocialPlatform, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(models.SocialPlatformTelegram):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_TELEGRAM, nil
	case string(models.SocialPlatformGithub):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_GITHUB, nil
	case string(models.SocialPlatformVK):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_VK, nil
	case string(models.SocialPlatformLinkedIn):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_LINKEDIN, nil
	case string(models.SocialPlatformX):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_X, nil
	case string(models.SocialPlatformYoutube):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_YOUTUBE, nil
	case string(models.SocialPlatformWebsite):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_WEBSITE, nil
	case string(models.SocialPlatformOther):
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_OTHER, nil
	default:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_UNSPECIFIED, fmt.Errorf("invalid social platform: %s", value)
	}
}
