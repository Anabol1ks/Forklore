package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	"github.com/gin-gonic/gin"
)

type StudyHandler struct {
	client *clients.StudyClient
}

func NewStudyHandler(client *clients.StudyClient) *StudyHandler {
	return &StudyHandler{client: client}
}

// GenerateText godoc
//
//	@Summary		Сгенерировать карточки или вопросы по тексту
//	@Description	Проксирует запрос во внутренний Python study-service
//	@Tags			study
//	@Accept			plain
//	@Produce		json
//	@Param			mode	query	string	true	"flashcards | random_questions"
//	@Param			count	query	int		false	"Количество элементов"	default(10)
//	@Param			body	body	string	true	"Raw note text"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		503		{object}	models.ErrorResponse
//	@Router			/study/generate-text [post]
func (h *StudyHandler) GenerateText(c *gin.Context) {
	mode := strings.TrimSpace(c.Query("mode"))
	if mode == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "mode query parameter is required",
		})
		return
	}

	count := 10
	if rawCount := strings.TrimSpace(c.Query("count")); rawCount != "" {
		parsedCount, err := strconv.Atoi(rawCount)
		if err != nil || parsedCount < 1 || parsedCount > 50 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "count must be an integer between 1 and 50",
			})
			return
		}
		count = parsedCount
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to read request body",
		})
		return
	}

	if strings.TrimSpace(string(body)) == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "request body must contain note text",
		})
		return
	}

	statusCode, payload, err := h.client.GenerateText(c.Request.Context(), mode, count, body)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Code:    http.StatusServiceUnavailable,
			Message: "study-service unavailable",
		})
		return
	}

	c.Data(statusCode, "application/json; charset=utf-8", payload)
}
