package repository

import (
	"context"
	"repository-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RepositoryStarRepository interface {
	Create(ctx context.Context, star *model.RepositoryStar) error
	DeleteByUserAndRepo(ctx context.Context, userID, repoID uuid.UUID) error
	Exists(ctx context.Context, userID, repoID uuid.UUID) (bool, error)
	CountByRepoID(ctx context.Context, repoID uuid.UUID) (int64, error)
	ListStarredRepositoriesByUser(ctx context.Context, userID uuid.UUID, params ListParams) ([]*model.Repository, int64, error)
}

type repositoryStarRepository struct {
	db *gorm.DB
}

func NewRepositoryStarRepository(db *gorm.DB) RepositoryStarRepository {
	return &repositoryStarRepository{db: db}
}

func (r *repositoryStarRepository) Create(ctx context.Context, star *model.RepositoryStar) error {
	return r.db.WithContext(ctx).Create(star).Error
}

func (r *repositoryStarRepository) DeleteByUserAndRepo(ctx context.Context, userID, repoID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND repo_id = ?", userID, repoID).
		Delete(&model.RepositoryStar{}).Error
}

func (r *repositoryStarRepository) Exists(ctx context.Context, userID, repoID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.RepositoryStar{}).
		Where("user_id = ? AND repo_id = ?", userID, repoID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repositoryStarRepository) CountByRepoID(ctx context.Context, repoID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.RepositoryStar{}).
		Where("repo_id = ?", repoID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repositoryStarRepository) ListStarredRepositoriesByUser(ctx context.Context, userID uuid.UUID, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.db.WithContext(ctx).
		Model(&model.Repository{}).
		Joins("JOIN repository_stars rs ON rs.repo_id = repositories.id").
		Where("rs.user_id = ?", userID).
		Where("repositories.deleted_at IS NULL").
		Where("repositories.visibility = ? OR repositories.owner_id = ?", model.RepositoryVisibilityPublic, userID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.db.WithContext(ctx).
		Model(&model.Repository{}).
		Preload("Tag").
		Joins("JOIN repository_stars rs ON rs.repo_id = repositories.id").
		Where("rs.user_id = ?", userID).
		Where("repositories.deleted_at IS NULL").
		Where("repositories.visibility = ? OR repositories.owner_id = ?", model.RepositoryVisibilityPublic, userID).
		Order("rs.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}
