package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentFormat string

const (
	DocumentFormatMarkdown DocumentFormat = "markdown"
)

type Document struct {
	ID                   uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RepoID               uuid.UUID      `gorm:"column:repo_id;type:uuid;not null;index"`
	AuthorID             uuid.UUID      `gorm:"column:author_id;type:uuid;not null;index"`
	Title                string         `gorm:"column:title;type:varchar(200);not null"`
	Slug                 string         `gorm:"column:slug;type:varchar(100);not null"`
	Format               DocumentFormat `gorm:"column:format;type:varchar(16);not null;default:'markdown'"`
	CurrentVersionID     *uuid.UUID     `gorm:"column:current_version_id;type:uuid"`
	LatestDraftUpdatedAt *time.Time     `gorm:"column:latest_draft_updated_at;type:timestamptz"`

	CreatedAt time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Draft          *DocumentDraft     `gorm:"foreignKey:DocumentID;references:ID;constraint:OnDelete:CASCADE"`
	CurrentVersion *DocumentVersion   `gorm:"foreignKey:CurrentVersionID;references:ID;constraint:OnDelete:SET NULL"`
	Versions       []*DocumentVersion `gorm:"foreignKey:DocumentID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Document) TableName() string {
	return "documents"
}
