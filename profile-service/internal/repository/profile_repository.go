package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"profile-service/internal/model"

	"github.com/google/uuid"
)

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(ctx context.Context, profile *model.Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *profileRepository) CreateOrIgnore(ctx context.Context, profile *model.Profile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).
		Create(profile).Error
}

func (r *profileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Profile, error) {
	var profile model.Profile

	err := r.queryWithRelations(ctx).
		Where("user_id = ?", userID).
		Take(&profile).Error
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func (r *profileRepository) GetByUsername(ctx context.Context, username string) (*model.Profile, error) {
	var profile model.Profile

	err := r.queryWithRelations(ctx).
		Where("username = ?", username).
		Take(&profile).Error
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *model.Profile) error {
	return r.db.WithContext(ctx).
		Omit(clause.Associations).
		Save(profile).Error
}

func (r *profileRepository) UpdateReadme(ctx context.Context, userID uuid.UUID, readme *string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Profile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"readme_markdown": readme,
			"updated_at":      gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *profileRepository) SetTitle(ctx context.Context, userID uuid.UUID, titleCode *string, titleSource model.ProfileTitleSource) error {
	result := r.db.WithContext(ctx).
		Model(&model.Profile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"title_code":   titleCode,
			"title_source": titleSource,
			"updated_at":   gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *profileRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Profile{})
}

func (r *profileRepository) queryWithRelations(ctx context.Context) *gorm.DB {
	return r.baseQuery(ctx).
		Preload("Title").
		Preload("SocialLinks", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC, created_at ASC")
		})
}
