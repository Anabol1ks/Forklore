package repository

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fileVersionRepository struct {
	db *gorm.DB
}

func NewFileVersionRepository(db *gorm.DB) FileVersionRepository {
	return &fileVersionRepository{db: db}
}

func (r *fileVersionRepository) Create(ctx context.Context, version *model.FileVersion) error {
	return r.db.WithContext(ctx).Create(version).Error
}

func (r *fileVersionRepository) GetByID(ctx context.Context, versionID uuid.UUID) (*model.FileVersion, error) {
	var version model.FileVersion

	err := r.baseQuery(ctx).
		Where("id = ?", versionID).
		Take(&version).Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *fileVersionRepository) GetByFileAndVersionID(ctx context.Context, fileID, versionID uuid.UUID) (*model.FileVersion, error) {
	var version model.FileVersion

	err := r.baseQuery(ctx).
		Where("file_id = ? AND id = ?", fileID, versionID).
		Take(&version).Error
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *fileVersionRepository) GetLatestVersionNumber(ctx context.Context, fileID uuid.UUID) (uint32, error) {
	type result struct {
		VersionNumber uint32
	}

	var row result
	err := r.db.WithContext(ctx).
		Model(&model.FileVersion{}).
		Select("version_number").
		Where("file_id = ?", fileID).
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

func (r *fileVersionRepository) ListByFileID(ctx context.Context, fileID uuid.UUID, params ListParams) ([]*model.FileVersion, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("file_id = ?", fileID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var versions []*model.FileVersion
	err := r.baseQuery(ctx).
		Where("file_id = ?", fileID).
		Order("version_number DESC").
		Limit(limit).
		Offset(offset).
		Find(&versions).Error
	if err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

func (r *fileVersionRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.FileVersion{})
}
