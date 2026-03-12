package handlers

import (
	"net/http"
	"strings"

	"api-gateway/internal/clients"
	"api-gateway/internal/models"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
)

type AuthHandler struct {
	client *clients.AuthClient
}

func NewAuthHandler(client *clients.AuthClient) *AuthHandler {
	return &AuthHandler{client: client}
}

// Register godoc
//
//	@Summary		Регистрация нового пользователя
//	@Description	Создаёт нового пользователя и возвращает токены аутентификации
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.RegisterRequest	true	"Данные регистрации"
//	@Success		201		{object}	models.AuthResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		409		{object}	models.ErrorResponse	"Username или email уже заняты"
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := h.client.Client.Register(c.Request.Context(), &authv1.RegisterRequest{
		Username:   req.Username,
		Email:      req.Email,
		Password:   req.Password,
		DeviceName: c.GetHeader("X-Device-Name"),
		UserAgent:  c.GetHeader("User-Agent"),
		Ip:         c.ClientIP(),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusCreated, mapAuthResponse(resp))
}

// Login godoc
//
//	@Summary		Вход в аккаунт
//	@Description	Аутентификация по логину (username или email) и паролю
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.LoginRequest	true	"Данные входа"
//	@Success		200		{object}	models.AuthResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse	"Неверные учётные данные"
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := h.client.Client.Login(c.Request.Context(), &authv1.LoginRequest{
		Login:      req.Login,
		Password:   req.Password,
		DeviceName: c.GetHeader("X-Device-Name"),
		UserAgent:  c.GetHeader("User-Agent"),
		Ip:         c.ClientIP(),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, mapAuthResponse(resp))
}

// Refresh godoc
//
//	@Summary		Обновление токенов
//	@Description	Обновляет пару токенов по refresh токену
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.RefreshRequest	true	"Refresh токен"
//	@Success		200		{object}	models.AuthResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse	"Невалидный или истёкший refresh токен"
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := h.client.Client.Refresh(c.Request.Context(), &authv1.RefreshRequest{
		RefreshToken: req.RefreshToken,
		DeviceName:   c.GetHeader("X-Device-Name"),
		UserAgent:    c.GetHeader("User-Agent"),
		Ip:           c.ClientIP(),
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, mapAuthResponse(resp))
}

// Logout godoc
//
//	@Summary		Выход из сессии
//	@Description	Удаляет конкретную сессию по refresh токену
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	models.LogoutRequest	true	"Refresh токен"
//	@Success		204		"No Content"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	_, err := h.client.Client.Logout(c.Request.Context(), &authv1.LogoutRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// LogoutAll godoc
//
//	@Summary		Выход из всех сессий
//	@Description	Удаляет все сессии текущего пользователя
//	@Tags			auth
//	@Produce		json
//	@Success		204		"No Content"
//	@Failure		401		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx := forwardAuth(c)

	_, err := h.client.Client.LogoutAll(ctx, &emptypb.Empty{})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMe godoc
//
//	@Summary		Получение текущего пользователя
//	@Description	Возвращает информацию о текущем авторизованном пользователе
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	models.GetMeResponse
//	@Failure		401	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	ctx := forwardAuth(c)

	resp, err := h.client.Client.GetMe(ctx, &emptypb.Empty{})
	if err != nil {
		code, errResp := handleGRPCError(err)
		c.JSON(code, errResp)
		return
	}

	c.JSON(http.StatusOK, models.GetMeResponse{
		User: mapUser(resp.GetUser()),
	})
}

// forwardAuth пробрасывает Authorization заголовок в gRPC metadata.
func forwardAuth(c *gin.Context) *gin.Context {
	token := c.GetHeader("Authorization")
	if token != "" {
		md := metadata.Pairs("authorization", token)
		ctx := metadata.NewOutgoingContext(c.Request.Context(), md)
		c.Request = c.Request.WithContext(ctx)
	}
	return c
}

// extractToken извлекает токен из Authorization заголовка.
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
