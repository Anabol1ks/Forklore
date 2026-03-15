package model

import (
	"time"

	"github.com/google/uuid"
)

type DocumentVersion struct {
	ID            uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	DocumentID    uuid.UUID `gorm:"column:document_id;type:uuid;not null;index"`
	AuthorID      uuid.UUID `gorm:"column:author_id;type:uuid;not null"`
	VersionNumber uint32    `gorm:"column:version_number;not null"`
	Content       string    `gorm:"column:content;type:text;not null"`
	ChangeSummary *string   `gorm:"column:change_summary;type:varchar(255)"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"`

	Document *Document `gorm:"foreignKey:DocumentID;references:ID;constraint:OnDelete:CASCADE"`
}

func (DocumentVersion) TableName() string {
	return "document_versions"
}
