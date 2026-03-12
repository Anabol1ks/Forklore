package models

// ──────────────── Requests ────────────────

// RegisterRequest represents registration data.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32" example:"john_doe"`
	Email    string `json:"email" binding:"required,email,max=254" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"secureP@ssw0rd"`
}

// LoginRequest represents login data.
type LoginRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=254" example:"john_doe"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"secureP@ssw0rd"`
}

// RefreshRequest represents token refresh data.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"dGhpcyBpcyBhIHJlZnJlc2g..."`
}

// LogoutRequest represents logout data.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"dGhpcyBpcyBhIHJlZnJlc2g..."`
}

// ──────────────── Responses ────────────────

// UserResponse represents user information.
type UserResponse struct {
	UserID      string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username    string  `json:"username" example:"john_doe"`
	Email       string  `json:"email" example:"john@example.com"`
	Role        string  `json:"role" example:"user" enums:"user,admin"`
	Status      string  `json:"status" example:"active" enums:"active,blocked,deleted"`
	CreatedAt   string  `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2026-01-01T00:00:00Z"`
	LastLoginAt *string `json:"last_login_at,omitempty" example:"2026-01-01T00:00:00Z"`
}

// TokenPairResponse represents authentication tokens.
type TokenPairResponse struct {
	AccessToken      string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken     string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJlc2g..."`
	TokenType        string `json:"token_type" example:"Bearer"`
	AccessExpiresAt  string `json:"access_expires_at" example:"2026-01-01T01:00:00Z"`
	RefreshExpiresAt string `json:"refresh_expires_at" example:"2026-01-08T00:00:00Z"`
	SessionID        string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// AuthResponse represents authentication response with user and tokens.
type AuthResponse struct {
	User   UserResponse      `json:"user"`
	Tokens TokenPairResponse `json:"tokens"`
}

// GetMeResponse represents current user info.
type GetMeResponse struct {
	User UserResponse `json:"user"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"invalid request"`
}
