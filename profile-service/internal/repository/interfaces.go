package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"profile-service/internal/model"
)

type ListParams struct {
	Limit  int
	Offset int
}

type ProfilePreview struct {
	UserID      uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
	TitleCode   *string
	TitleLabel  *string
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *model.Profile) error
	CreateOrIgnore(ctx context.Context, profile *model.Profile) error

	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Profile, error)
	GetByUsername(ctx context.Context, username string) (*model.Profile, error)

	Update(ctx context.Context, profile *model.Profile) error
	UpdateReadme(ctx context.Context, userID uuid.UUID, readme *string) error
	SetTitle(ctx context.Context, userID uuid.UUID, titleCode *string, titleSource model.ProfileTitleSource) error
}

type SocialLinkRepository interface {
	Create(ctx context.Context, link *model.ProfileSocialLink) error
	GetByID(ctx context.Context, socialLinkID uuid.UUID) (*model.ProfileSocialLink, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.ProfileSocialLink, error)
	Update(ctx context.Context, link *model.ProfileSocialLink) error
	DeleteByID(ctx context.Context, socialLinkID uuid.UUID) error
}

type FollowRepository interface {
	Follow(ctx context.Context, follow *model.ProfileFollow) error
	Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error
	IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)

	CountFollowers(ctx context.Context, userID uuid.UUID) (int64, error)
	CountFollowing(ctx context.Context, userID uuid.UUID) (int64, error)

	ListFollowers(ctx context.Context, userID uuid.UUID, params ListParams) ([]*ProfilePreview, int64, error)
	ListFollowing(ctx context.Context, userID uuid.UUID, params ListParams) ([]*ProfilePreview, int64, error)
}

type TitleRepository interface {
	GetByCode(ctx context.Context, code string) (*model.ProfileTitle, error)
	ListActive(ctx context.Context) ([]*model.ProfileTitle, error)
}

type Repository struct {
	db *gorm.DB

	Profile    ProfileRepository
	SocialLink SocialLinkRepository
	Follow     FollowRepository
	Title      TitleRepository
}
