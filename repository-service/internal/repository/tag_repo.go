package repository

import (
	"context"
	"repository-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TagRepository interface {
	GetByID(ctx context.Context, tagID uuid.UUID) (*model.RepositoryTag, error)
	ListActive(ctx context.Context) ([]*model.RepositoryTag, error)
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) GetByID(ctx context.Context, tagID uuid.UUID) (*model.RepositoryTag, error) {
	var tag model.RepositoryTag

	err := r.db.WithContext(ctx).
		Model(&model.RepositoryTag{}).
		Where("id = ?", tagID).
		Take(&tag).Error
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *tagRepository) ListActive(ctx context.Context) ([]*model.RepositoryTag, error) {
	var tags []*model.RepositoryTag

	err := r.db.WithContext(ctx).
		Model(&model.RepositoryTag{}).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&tags).Error
	if err != nil {
		return nil, err
	}

	return tags, nil
}
