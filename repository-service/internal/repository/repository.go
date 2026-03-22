package repository

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB

	Repo RepoRepository
	Tag  TagRepository
	Star RepositoryStarRepository
}

type ListParams struct {
	Limit  int
	Offset int
}

func New(db *gorm.DB) *Repository {
	return &Repository{
		db:   db,
		Repo: NewRepoRepository(db),
		Tag:  NewTagRepository(db),
		Star: NewRepositoryStarRepository(db),
	}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	return New(tx)
}

func (r *Repository) Transaction(ctx context.Context, fn func(repo *Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(r.WithTx(tx))
	})
}
