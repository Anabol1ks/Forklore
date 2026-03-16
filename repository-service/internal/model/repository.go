package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RepositoryVisibility string

const (
	RepositoryVisibilityPublic  RepositoryVisibility = "public"
	RepositoryVisibilityPrivate RepositoryVisibility = "private"
)

type RepositoryType string

const (
	RepositoryTypeArticle RepositoryType = "article"
	RepositoryTypeNotes   RepositoryType = "notes"
	RepositoryTypeMixed   RepositoryType = "mixed"
)

type Repository struct {
	ID            uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID       uuid.UUID            `gorm:"type:uuid;not null;index"`
	OwnerUsername string               `gorm:"type:varchar(32);index"`
	TagID         uuid.UUID            `gorm:"type:uuid;not null;index"`
	Name          string               `gorm:"type:varchar(100);not null"`
	Slug          string               `gorm:"type:varchar(64);not null"`
	Description   *string              `gorm:"type:text"`
	Visibility    RepositoryVisibility `gorm:"type:varchar(16);not null;default:'private'"`
	Type          RepositoryType       `gorm:"type:varchar(16);not null;default:'mixed'"`
	ParentRepoID  *uuid.UUID           `gorm:"type:uuid;index"`
	ParentRepo    *Repository          `gorm:"foreignKey:ParentRepoID;references:ID;constraint:OnDelete:SET NULL"`
	Tag           *RepositoryTag       `gorm:"foreignKey:TagID;references:ID"`

	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Repository) TableName() string {
	return "repositories"
}
