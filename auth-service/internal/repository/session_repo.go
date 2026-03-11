package repository

import (
	model "auth-service/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionRepo interface {
	Create(ctx context.Context, session *model.RefreshSession) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshSession, error)
	RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type sessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) SessionRepo {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.RefreshSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshSession, error) {
	var session model.RefreshSession

	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		Take(&session).Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *sessionRepo) RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&model.RefreshSession{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]any{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *sessionRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		}).Error
}
