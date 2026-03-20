package repository

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type fileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *fileRepository) GetByID(ctx context.Context, fileID uuid.UUID) (*model.File, error) {
	var file model.File

	err := r.queryWithRelations(ctx).
		Where("id = ?", fileID).
		Take(&file).Error
	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (r *fileRepository) GetByIDForUpdate(ctx context.Context, fileID uuid.UUID) (*model.File, error) {
	var file model.File

	err := r.baseQuery(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", fileID).
		Take(&file).Error
	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (r *fileRepository) Update(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).
		Omit(clause.Associations).
		Save(file).Error
}

func (r *fileRepository) DeleteByID(ctx context.Context, fileID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", fileID).
		Delete(&model.File{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *fileRepository) ListByRepoID(ctx context.Context, repoID uuid.UUID, params ListParams) ([]*model.File, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("repo_id = ?", repoID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []*model.File
	err := r.baseQuery(ctx).
		Where("repo_id = ?", repoID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&files).Error
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

func (r *fileRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.File{})
}

func (r *fileRepository) queryWithRelations(ctx context.Context) *gorm.DB {
	return r.baseQuery(ctx).
		Preload("CurrentVersion")
}
