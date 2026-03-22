package repository

import (
	model "auth-service/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	List(ctx context.Context) ([]*model.User, error)
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID, ts time.Time) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).Where("email = ?", email).Take(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).Where("username = ?", username).Take(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
func (r *userRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", login, login).
		Take(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) List(ctx context.Context) ([]*model.User, error) {
	users := make([]*model.User, 0)

	err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Order("id ASC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepo) UpdateLastLoginAt(ctx context.Context, userID uuid.UUID, ts time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"last_login_at": ts,
			"updated_at":    ts,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
