package repository

import (
	"context"
	"profile-service/internal/model"

	"gorm.io/gorm"
)

type titleRepository struct {
	db *gorm.DB
}

func NewTitleRepository(db *gorm.DB) TitleRepository {
	return &titleRepository{db: db}
}

func (r *titleRepository) GetByCode(ctx context.Context, code string) (*model.ProfileTitle, error) {
	var title model.ProfileTitle

	err := r.db.WithContext(ctx).
		Model(&model.ProfileTitle{}).
		Where("code = ?", code).
		Take(&title).Error
	if err != nil {
		return nil, err
	}

	return &title, nil
}

func (r *titleRepository) ListActive(ctx context.Context) ([]*model.ProfileTitle, error) {
	var titles []*model.ProfileTitle

	err := r.db.WithContext(ctx).
		Model(&model.ProfileTitle{}).
		Where("is_active = ?", true).
		Order("sort_order ASC, label ASC").
		Find(&titles).Error
	if err != nil {
		return nil, err
	}

	return titles, nil
}
