package repository

import (
	"context"
	"repository-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RepoRepository interface {
	Create(ctx context.Context, repo *model.Repository) error
	GetByID(ctx context.Context, repoID uuid.UUID) (*model.Repository, error)
	GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.Repository, error)
	GetByOwnerUsernameAndSlug(ctx context.Context, ownerUsername string, slug string) (*model.Repository, error)
	Update(ctx context.Context, repo *model.Repository) error
	DeleteByID(ctx context.Context, repoID uuid.UUID) error

	ListByOwner(ctx context.Context, ownerID uuid.UUID, params ListParams) ([]*model.Repository, int64, error)
	ListByOwnerUsername(ctx context.Context, ownerUsername string, params ListParams) ([]*model.Repository, int64, error)
	ListPublicByOwner(ctx context.Context, ownerID uuid.UUID, params ListParams) ([]*model.Repository, int64, error)
	ListPublicByOwnerUsername(ctx context.Context, ownerUsername string, params ListParams) ([]*model.Repository, int64, error)
	ListForks(ctx context.Context, parentRepoID uuid.UUID, params ListParams) ([]*model.Repository, int64, error)
}

type repoRepository struct {
	db *gorm.DB
}

func NewRepoRepository(db *gorm.DB) RepoRepository {
	return &repoRepository{db: db}
}

func (r *repoRepository) Create(ctx context.Context, repo *model.Repository) error {
	return r.db.WithContext(ctx).Create(repo).Error
}

func (r *repoRepository) GetByID(ctx context.Context, repoID uuid.UUID) (*model.Repository, error) {
	var repo model.Repository

	err := r.queryWithTag(ctx).
		Where("id = ?", repoID).
		Take(&repo).Error
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func (r *repoRepository) GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.Repository, error) {
	var repo model.Repository

	err := r.queryWithTag(ctx).
		Where("owner_id = ? AND slug = ?", ownerID, slug).
		Take(&repo).Error
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func (r *repoRepository) GetByOwnerUsernameAndSlug(ctx context.Context, ownerUsername string, slug string) (*model.Repository, error) {
	var repo model.Repository

	err := r.queryWithTag(ctx).
		Where("owner_username = ? AND slug = ?", ownerUsername, slug).
		Take(&repo).Error
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func (r *repoRepository) Update(ctx context.Context, repo *model.Repository) error {
	return r.db.WithContext(ctx).
		Omit(clause.Associations).
		Save(repo).Error
}

func (r *repoRepository) DeleteByID(ctx context.Context, repoID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", repoID).
		Delete(&model.Repository{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *repoRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("owner_id = ?", ownerID)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.queryWithTag(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repoRepository) ListByOwnerUsername(ctx context.Context, ownerUsername string, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("owner_username = ?", ownerUsername)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.queryWithTag(ctx).
		Where("owner_username = ?", ownerUsername).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repoRepository) ListPublicByOwner(ctx context.Context, ownerID uuid.UUID, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("owner_id = ? AND visibility = ?", ownerID, model.RepositoryVisibilityPublic)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.queryWithTag(ctx).
		Where("owner_id = ? AND visibility = ?", ownerID, model.RepositoryVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repoRepository) ListPublicByOwnerUsername(ctx context.Context, ownerUsername string, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	countQuery := r.baseQuery(ctx).
		Where("owner_username = ? AND visibility = ?", ownerUsername, model.RepositoryVisibilityPublic)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.queryWithTag(ctx).
		Where("owner_username = ? AND visibility = ?", ownerUsername, model.RepositoryVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repoRepository) ListForks(ctx context.Context, parentRepoID uuid.UUID, params ListParams) ([]*model.Repository, int64, error) {
	limit, offset := normalizePagination(params)

	// Для MVP по умолчанию показываем только публичные fork'и,
	// чтобы не светить приватные репозитории.
	countQuery := r.baseQuery(ctx).
		Where("parent_repo_id = ? AND visibility = ?", parentRepoID, model.RepositoryVisibilityPublic)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	err := r.queryWithTag(ctx).
		Where("parent_repo_id = ? AND visibility = ?", parentRepoID, model.RepositoryVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error
	if err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repoRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Repository{})
}

func (r *repoRepository) queryWithTag(ctx context.Context) *gorm.DB {
	return r.baseQuery(ctx).Preload("Tag")
}

func normalizePagination(params ListParams) (limit int, offset int) {
	limit = params.Limit
	offset = params.Offset

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
