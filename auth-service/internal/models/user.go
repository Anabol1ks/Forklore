package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type UserStatus string

const (
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
	UserStatusDeleted UserStatus = "deleted"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string     `gorm:"type:varchar(32);not null;uniqueIndex"`
	Email        string     `gorm:"type:varchar(254);not null;uniqueIndex"`
	PasswordHash string     `gorm:"type:text;not null"`
	Role         UserRole   `gorm:"type:varchar(16);not null;default:'user'"`
	Status       UserStatus `gorm:"type:varchar(16);not null;default:'active'"`
	LastLoginAt  *time.Time `gorm:"type:timestamptz"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`

	Sessions []RefreshSession `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (User) TableName() string {
	return "users"
}
