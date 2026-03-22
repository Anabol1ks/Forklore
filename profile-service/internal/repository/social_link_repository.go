package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"profile-service/internal/model"

	"github.com/google/uuid"
)

type socialLinkRepository struct {
	db *gorm.DB
}

func NewSocialLinkRepository(db *gorm.DB) SocialLinkRepository {
	return &socialLinkRepository{db: db}
}

func (r *socialLinkRepository) Create(ctx context.Context, link *model.ProfileSocialLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *socialLinkRepository) GetByID(ctx context.Context, socialLinkID uuid.UUID) (*model.ProfileSocialLink, error) {
	var link model.ProfileSocialLink

	err := r.db.WithContext(ctx).
		Model(&model.ProfileSocialLink{}).
		Where("id = ?", socialLinkID).
		Take(&link).Error
	if err != nil {
		return nil, err
	}

	return &link, nil
}

func (r *socialLinkRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.ProfileSocialLink, error) {
	var links []*model.ProfileSocialLink

	err := r.db.WithContext(ctx).
		Model(&model.ProfileSocialLink{}).
		Where("user_id = ?", userID).
		Order("position ASC, created_at ASC").
		Find(&links).Error
	if err != nil {
		return nil, err
	}

	return links, nil
}

func (r *socialLinkRepository) Update(ctx context.Context, link *model.ProfileSocialLink) error {
	return r.db.WithContext(ctx).
		Omit(clause.Associations).
		Save(link).Error
}

func (r *socialLinkRepository) DeleteByID(ctx context.Context, socialLinkID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", socialLinkID).
		Delete(&model.ProfileSocialLink{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
