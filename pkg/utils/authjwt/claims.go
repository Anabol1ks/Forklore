package authjwt

import (
	"time"

	"github.com/google/uuid"
)

type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Username  string
	Email     string
	Role      string
	Status    string
	ExpiresAt time.Time
}
