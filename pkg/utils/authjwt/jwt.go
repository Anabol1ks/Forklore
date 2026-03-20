package authjwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type jwtClaims struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type TokenVerifier interface {
	ParseAccessToken(token string) (*AccessClaims, error)
}

type JWTVerifier struct {
	secret []byte
}

func NewJWTVerifier(secret string) *JWTVerifier {
	return &JWTVerifier{
		secret: []byte(secret),
	}
}

func (v *JWTVerifier) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidAccessToken
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		return nil, ErrInvalidAccessToken
	}

	return &AccessClaims{
		UserID:    userID,
		SessionID: sessionID,
		Username:  claims.Username,
		Email:     claims.Email,
		Role:      claims.Role,
		Status:    claims.Status,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
