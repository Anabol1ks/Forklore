package repository

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type documentDraftRepository struct {
	db *gorm.DB
}

func NewDocumentDraftRepository(db *gorm.DB) DocumentDraftRepository {
	return &documentDraftRepository{db: db}
}

func (r *documentDraftRepository) Create(ctx context.Context, draft *model.DocumentDraft) error {
	return r.db.WithContext(ctx).Create(draft).Error
}

func (r *documentDraftRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID) (*model.DocumentDraft, error) {
	var draft model.DocumentDraft

	err := r.db.WithContext(ctx).
		Model(&model.DocumentDraft{}).
		Where("document_id = ?", documentID).
		Take(&draft).Error
	if err != nil {
		return nil, err
	}

	return &draft, nil
}

func (r *documentDraftRepository) Upsert(ctx context.Context, draft *model.DocumentDraft) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "document_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"content":    draft.Content,
				"updated_by": draft.UpdatedBy,
				"updated_at": draft.UpdatedAt,
			}),
		}).
		Create(draft).Error
}

func (r *documentDraftRepository) DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Delete(&model.DocumentDraft{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
