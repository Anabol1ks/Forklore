package model

import (
	"time"

	"github.com/google/uuid"
)

type RepositoryStar struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	RepoID uuid.UUID `gorm:"type:uuid;not null;index"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`

	Repository *Repository `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
}

func (RepositoryStar) TableName() string {
	return "repository_stars"
}
