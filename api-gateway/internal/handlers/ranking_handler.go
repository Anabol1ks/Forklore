package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	rankingv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/v1"
	"github.com/gin-gonic/gin"
)

type RankingHandler struct {
	client *clients.RankingClient
}

func NewRankingHandler(client *clients.RankingClient) *RankingHandler {
	return &RankingHandler{client: client}
}

// GetOverallLeaderboard godoc
//
//	@Summary		Получить общий рейтинг
//	@Description	Возвращает таблицу лидеров по общему рейтингу пользователей
//	@Tags			rankings
//	@Produce		json
//	@Param			limit	query		int	false	"Лимит" default(20)
//	@Param			offset	query		int	false	"Смещение" default(0)
//	@Success		200		{object}	models.RankingResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/rankings/overall [get]
func (h *RankingHandler) GetOverallLeaderboard(c *gin.Context) {
	limit, offset, err := parseRankingPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	resp, callErr := h.client.Client.GetOverallLeaderboard(forwardAuth(c), &rankingv1.GetLeaderboardRequest{
		Limit:  uint32(limit),
		Offset: uint32(offset),
	})
	if callErr != nil {
		code, errResp := handleGRPCError(callErr)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, mapRankingResponse(resp))
}

// GetMonthlyLeaderboard godoc
//
//	@Summary		Получить месячный рейтинг
//	@Description	Возвращает таблицу лидеров по рейтингу за последние 30 дней
//	@Tags			rankings
//	@Produce		json
//	@Param			limit	query		int	false	"Лимит" default(20)
//	@Param			offset	query		int	false	"Смещение" default(0)
//	@Success		200		{object}	models.RankingResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/rankings/monthly [get]
func (h *RankingHandler) GetMonthlyLeaderboard(c *gin.Context) {
	limit, offset, err := parseRankingPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	resp, callErr := h.client.Client.GetMonthlyLeaderboard(forwardAuth(c), &rankingv1.GetLeaderboardRequest{
		Limit:  uint32(limit),
		Offset: uint32(offset),
	})
	if callErr != nil {
		code, errResp := handleGRPCError(callErr)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, mapRankingResponse(resp))
}

// GetSubjectLeaderboard godoc
//
//	@Summary		Получить предметный рейтинг
//	@Description	Возвращает таблицу лидеров по выбранному предметному тегу
//	@Tags			rankings
//	@Produce		json
//	@Param			tag_id	path		string	true	"ID предметного тега"
//	@Param			limit	query		int		false	"Лимит" default(20)
//	@Param			offset	query		int		false	"Смещение" default(0)
//	@Success		200		{object}	models.RankingResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/rankings/subject/{tag_id} [get]
func (h *RankingHandler) GetSubjectLeaderboard(c *gin.Context) {
	tagID := strings.TrimSpace(c.Param("tag_id"))
	if tagID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: "tag_id is required"})
		return
	}

	limit, offset, err := parseRankingPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	resp, callErr := h.client.Client.GetSubjectLeaderboard(forwardAuth(c), &rankingv1.GetSubjectLeaderboardRequest{
		TagId:  &commonv1.UUID{Value: tagID},
		Limit:  uint32(limit),
		Offset: uint32(offset),
	})
	if callErr != nil {
		code, errResp := handleGRPCError(callErr)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, mapRankingResponse(resp))
}

func parseRankingPagination(c *gin.Context) (int, int, error) {
	limit := 20
	offset := 0

	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit")
		}
		limit = parsed
	}

	if rawOffset := strings.TrimSpace(c.Query("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset")
		}
		offset = parsed
	}

	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must be greater than or equal to 0")
	}

	return limit, offset, nil
}

func mapRankingResponse(resp *rankingv1.GetLeaderboardResponse) models.RankingResponse {
	if resp == nil {
		return models.RankingResponse{Entries: []models.RankingEntryResponse{}, Total: 0}
	}

	entries := make([]models.RankingEntryResponse, 0, len(resp.GetEntries()))
	for _, item := range resp.GetEntries() {
		entry := models.RankingEntryResponse{
			UserID:                  item.GetUserId().GetValue(),
			Username:                item.GetUsername(),
			DisplayName:             item.GetDisplayName(),
			AvatarURL:               item.GetAvatarUrl(),
			TitleLabel:              item.GetTitleLabel(),
			Score:                   item.GetScore(),
			FollowersCount:          item.GetFollowersCount(),
			FollowersGained30d:      item.GetFollowersGained_30D(),
			StarsReceivedTotal:      item.GetStarsReceivedTotal(),
			StarsReceived30d:        item.GetStarsReceived_30D(),
			ForksReceivedTotal:      item.GetForksReceivedTotal(),
			ForksReceived30d:        item.GetForksReceived_30D(),
			PublicRepositoriesCount: item.GetPublicRepositoriesCount(),
			ActivityPointsTotal:     item.GetActivityPointsTotal(),
			ActivityPoints30d:       item.GetActivityPoints_30D(),
			ActiveWeeksLast8:        item.GetActiveWeeksLast_8(),
			ActiveMonthsCount:       item.GetActiveMonthsCount(),
			SubjectScore:            item.GetSubjectScore(),
		}
		if item.GetTagId() != nil {
			entry.TagID = item.GetTagId().GetValue()
		}
		entries = append(entries, entry)
	}

	return models.RankingResponse{Entries: entries, Total: resp.GetTotal()}
}
