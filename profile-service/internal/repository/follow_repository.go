package repository

import (
	"context"
	"profile-service/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"
)

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(ctx context.Context, follow *model.ProfileFollow) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "follower_id"}, {Name: "followee_id"}},
			DoNothing: true,
		}).
		Create(follow).Error
}

func (r *followRepository) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&model.ProfileFollow{}).Error
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProfileFollow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followRepository) CountFollowers(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProfileFollow{}).
		Where("followee_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followRepository) CountFollowing(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProfileFollow{}).
		Where("follower_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followRepository) ListFollowers(ctx context.Context, userID uuid.UUID, params ListParams) ([]*ProfilePreview, int64, error) {
	limit, offset := normalizePagination(params)

	countSQL := `
SELECT COUNT(*)
FROM profile_follows pf
WHERE pf.followee_id = ?
`
	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, userID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
SELECT
    p.user_id,
    p.username,
    p.display_name,
    p.avatar_url,
    p.title_code,
    pt.label AS title_label
FROM profile_follows pf
JOIN profiles p ON p.user_id = pf.follower_id
LEFT JOIN profile_titles_catalog pt ON pt.code = p.title_code
WHERE pf.followee_id = ?
ORDER BY pf.created_at DESC
LIMIT ? OFFSET ?
`
	var result []*ProfilePreview
	if err := r.db.WithContext(ctx).Raw(querySQL, userID, limit, offset).Scan(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *followRepository) ListFollowing(ctx context.Context, userID uuid.UUID, params ListParams) ([]*ProfilePreview, int64, error) {
	limit, offset := normalizePagination(params)

	countSQL := `
SELECT COUNT(*)
FROM profile_follows pf
WHERE pf.follower_id = ?
`
	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, userID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
SELECT
    p.user_id,
    p.username,
    p.display_name,
    p.avatar_url,
    p.title_code,
    pt.label AS title_label
FROM profile_follows pf
JOIN profiles p ON p.user_id = pf.followee_id
LEFT JOIN profile_titles_catalog pt ON pt.code = p.title_code
WHERE pf.follower_id = ?
ORDER BY pf.created_at DESC
LIMIT ? OFFSET ?
`
	var result []*ProfilePreview
	if err := r.db.WithContext(ctx).Raw(querySQL, userID, limit, offset).Scan(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}
