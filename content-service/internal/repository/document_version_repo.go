package repository

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type documentVersionRepository struct {
	db *gorm.DB
}

func NewDocumentVersionRepository(db *gorm.DB) DocumentVersionRepository {
	return &documentVersionRepository{db: db}
}

func (r *documentVersionRepository) Create(ctx context.Context, version *model.DocumentVersion) error {
	return r.db.WithContext(ctx).Create(version).Error
}

func (r *documentVersionRepository) GetByID(ctx context.Context, versionID uuid.UUID) (*model.DocumentVersion, error) {
	var version model.DocumentVersion

	err := r.baseQuery(ctx).
		Where("id = ?", versionID).
		Take(&version).Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *documentVersionRepository) GetByDocumentAndVersionID(ctx context.Context, documentID, versionID uuid.UUID) (*model.DocumentVersion, error) {
	var version model.DocumentVersion

	err := r.baseQuery(ctx).
		Where("document_id = ? AND id = ?", documentID, versionID).
		Take(&version).Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *documentVersionRepository) GetLatestVersionNumber(ctx context.Context, documentID uuid.UUID) (uint32, error) {
	type result struct {
		VersionNumber uint32
	}

	var row result
	err := r.db.WithContext(ctx).
		Model(&model.DocumentVersion{}).
		Select("version_number").
		Where("document_id = ?", documentID).
		Order("version_number DESC").
		Limit(1).
		Take(&row).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}

	return row.VersionNumber, nil
}

func (r *documentVersionRepository) ListByDocumentID(ctx context.Context, documentID uuid.UUID, params ListParams) ([]*model.DocumentVersion, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("document_id = ?", documentID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var versions []*model.DocumentVersion
	err := r.baseQuery(ctx).
		Where("document_id = ?", documentID).
		Order("version_number DESC").
		Limit(limit).
		Offset(offset).
		Find(&versions).Error
	if err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

func (r *documentVersionRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.DocumentVersion{})
}
