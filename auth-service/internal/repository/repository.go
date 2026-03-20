package repository

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB

	User    UserRepo
	Session SessionRepo
}

func buildRepository(db *gorm.DB) *Repository {
	return &Repository{
		db:      db,
		User:    NewUserRepo(db),
		Session: NewSessionRepo(db),
	}
}

func New(db *gorm.DB) *Repository {
	return buildRepository(db)
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
