package repository

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type documentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(ctx context.Context, document *model.Document) error {
	return r.db.WithContext(ctx).Create(document).Error
}

func (r *documentRepository) GetByID(ctx context.Context, documentID uuid.UUID) (*model.Document, error) {
	var document model.Document

	err := r.queryWithRelations(ctx).
		Where("id = ?", documentID).
		Take(&document).Error
	if err != nil {
		return nil, err
	}

	return &document, nil
}

func (r *documentRepository) GetByIDForUpdate(ctx context.Context, documentID uuid.UUID) (*model.Document, error) {
	var document model.Document

	err := r.baseQuery(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", documentID).
		Take(&document).Error
	if err != nil {
		return nil, err
	}

	return &document, nil
}

func (r *documentRepository) GetByRepoAndSlug(ctx context.Context, repoID uuid.UUID, slug string) (*model.Document, error) {
	var document model.Document

	err := r.queryWithRelations(ctx).
		Where("repo_id = ? AND slug = ?", repoID, slug).
		Take(&document).Error
	if err != nil {
		return nil, err
	}

	return &document, nil
}

func (r *documentRepository) Update(ctx context.Context, document *model.Document) error {
	return r.db.WithContext(ctx).
		Omit(clause.Associations).
		Save(document).Error
}

func (r *documentRepository) DeleteByID(ctx context.Context, documentID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", documentID).
		Delete(&model.Document{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *documentRepository) ListByRepoID(ctx context.Context, repoID uuid.UUID, params ListParams) ([]*model.Document, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("repo_id = ?", repoID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var documents []*model.Document
	err := r.baseQuery(ctx).
		Where("repo_id = ?", repoID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error
	if err != nil {
		return nil, 0, err
	}

	return documents, total, nil
}

func (r *documentRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Document{})
}

func (r *documentRepository) queryWithRelations(ctx context.Context) *gorm.DB {
	return r.baseQuery(ctx).
		Preload("Draft").
		Preload("CurrentVersion")
}
