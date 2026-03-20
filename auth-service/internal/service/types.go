package service

import (
	model "auth-service/internal/models"
	"time"

	"github.com/google/uuid"
)

type ClientMeta struct {
	DeviceName string
	UserAgent  string
	IP         string
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
	Meta     ClientMeta
}

type LoginInput struct {
	Login    string
	Password string
	Meta     ClientMeta
}

type RefreshInput struct {
	RefreshToken string
	Meta         ClientMeta
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	SessionID        uuid.UUID
}

type AuthOutput struct {
	User   *model.User
	Tokens TokenPair
}

type IntrospectOutput struct {
	Active    bool
	UserID    uuid.UUID
	SessionID uuid.UUID
	Username  string
	Email     string
	Role      model.UserRole
	Status    model.UserStatus
	ExpiresAt time.Time
}
