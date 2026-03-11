package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshSession struct {
	ID                   uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID               uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash            string     `gorm:"type:varchar(128);not null;uniqueIndex"`
	DeviceName           *string    `gorm:"type:varchar(128)"`
	UserAgent            *string    `gorm:"type:text"`
	IP                   *string    `gorm:"type:inet"`
	ExpiresAt            time.Time  `gorm:"type:timestamptz;not null;index"`
	RevokedAt            *time.Time `gorm:"type:timestamptz;index"`
	RotatedFromSessionID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt            time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt            time.Time  `gorm:"type:timestamptz;not null;default:now()"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (RefreshSession) TableName() string {
	return "refresh_sessions"
}
