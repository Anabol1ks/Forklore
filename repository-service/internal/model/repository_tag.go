package model

import (
	"time"

	"github.com/google/uuid"
)

type RepositoryTag struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	Slug        string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	Description *string   `gorm:"type:text"`
	IsActive    bool      `gorm:"type:boolean;not null;default:true"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (RepositoryTag) TableName() string {
	return "repository_tags"
}
