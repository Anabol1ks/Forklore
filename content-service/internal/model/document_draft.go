package model

import (
	"time"

	"github.com/google/uuid"
)

type DocumentDraft struct {
	DocumentID uuid.UUID `gorm:"column:document_id;type:uuid;primaryKey"`
	Content    string    `gorm:"column:content;type:text;not null;default:''"`
	UpdatedBy  uuid.UUID `gorm:"column:updated_by;type:uuid;not null"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()"`

	Document *Document `gorm:"foreignKey:DocumentID;references:ID;constraint:OnDelete:CASCADE"`
}

func (DocumentDraft) TableName() string {
	return "document_drafts"
}
