package repository

import "gorm.io/gorm"

func New(db *gorm.DB) *Repository {
	return &Repository{
		db:     db,
		Search: NewSearchIndexRepository(db),
	}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}
