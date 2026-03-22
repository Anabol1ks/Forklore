package service

import (
	"context"

	"github.com/google/uuid"

	"profile-service/internal/model"
)

type Pagination struct {
	Limit  int
	Offset int
}

type ProfileState struct {
	Profile        *model.Profile
	FollowersCount int64
	FollowingCount int64
}

type ProfilePreview struct {
	UserID      uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
	TitleCode   *string
	TitleLabel  *string
}

type CreateProfileInput struct {
	UserID   uuid.UUID
	Username string
}

type UpdateProfileInput struct {
	RequesterID uuid.UUID
	DisplayName string
	Bio         string
	AvatarURL   string
	CoverURL    string
	Location    string
	WebsiteURL  string
	IsPublic    bool
}

type UpdateProfileReadmeInput struct {
	RequesterID    uuid.UUID
	ReadmeMarkdown string
}

type UpsertProfileSocialLinkInput struct {
	RequesterID  uuid.UUID
	SocialLinkID *uuid.UUID
	Platform     model.SocialPlatform
	URL          string
	Label        string
	Position     int32
	IsVisible    bool
}

type SetProfileTitleInput struct {
	RequesterID uuid.UUID
	TitleCode   string
}

type ProfileService interface {
	CreateProfileIfNotExists(ctx context.Context, input CreateProfileInput) error

	GetMyProfile(ctx context.Context, requesterID uuid.UUID) (*ProfileState, error)
	GetProfileByUserID(ctx context.Context, requesterID, userID uuid.UUID) (*ProfileState, error)
	GetProfileByUsername(ctx context.Context, requesterID uuid.UUID, username string) (*ProfileState, error)

	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*ProfileState, error)
	UpdateProfileReadme(ctx context.Context, input UpdateProfileReadmeInput) (*ProfileState, error)

	ListProfileSocialLinks(ctx context.Context, requesterID, userID uuid.UUID) ([]*model.ProfileSocialLink, error)
	UpsertProfileSocialLink(ctx context.Context, input UpsertProfileSocialLinkInput) (*model.ProfileSocialLink, error)
	DeleteProfileSocialLink(ctx context.Context, requesterID, socialLinkID uuid.UUID) error

	FollowUser(ctx context.Context, followerID, followeeID uuid.UUID) error
	UnfollowUser(ctx context.Context, followerID, followeeID uuid.UUID) error
	ListFollowers(ctx context.Context, requesterID, userID uuid.UUID, pagination Pagination) ([]*ProfilePreview, int64, error)
	ListFollowing(ctx context.Context, requesterID, userID uuid.UUID, pagination Pagination) ([]*ProfilePreview, int64, error)

	ListAvailableTitles(ctx context.Context) ([]*model.ProfileTitle, error)
	SetProfileTitle(ctx context.Context, input SetProfileTitleInput) (*ProfileState, error)
}
