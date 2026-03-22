package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"profile-service/internal/domain"
	"profile-service/internal/model"
	"profile-service/internal/repository"
)

type profileService struct {
	repos            *repository.Repository
	defaultTitleCode string
}

func NewProfileService(repos *repository.Repository, defaultTitleCode string) ProfileService {
	defaultTitleCode = strings.TrimSpace(defaultTitleCode)
	if defaultTitleCode == "" {
		defaultTitleCode = "comer"
	}

	return &profileService{
		repos:            repos,
		defaultTitleCode: defaultTitleCode,
	}
}

func (s *profileService) CreateProfileIfNotExists(ctx context.Context, input CreateProfileInput) error {
	if input.UserID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	username := normalizeUsername(input.Username)
	if username == "" {
		return domain.ErrInvalidUsername
	}

	var titleCode *string
	if title, err := s.repos.Title.GetByCode(ctx, s.defaultTitleCode); err == nil && title.IsActive {
		code := title.Code
		titleCode = &code
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	profile := &model.Profile{
		UserID:      input.UserID,
		Username:    username,
		DisplayName: username,
		TitleCode:   titleCode,
		TitleSource: model.ProfileTitleSourceSystem,
		IsPublic:    true,
	}

	return s.repos.Profile.CreateOrIgnore(ctx, profile)
}

func (s *profileService) GetMyProfile(ctx context.Context, requesterID uuid.UUID) (*ProfileState, error) {
	if requesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	return s.getProfileByUserIDInternal(ctx, requesterID, requesterID)
}

func (s *profileService) GetProfileByUserID(ctx context.Context, requesterID, userID uuid.UUID) (*ProfileState, error) {
	return s.getProfileByUserIDInternal(ctx, requesterID, userID)
}

func (s *profileService) GetProfileByUsername(ctx context.Context, requesterID uuid.UUID, username string) (*ProfileState, error) {
	username = normalizeUsername(username)
	if username == "" {
		return nil, domain.ErrInvalidUsername
	}

	profile, err := s.repos.Profile.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	if !canReadProfile(requesterID, profile) {
		return nil, domain.ErrProfileAccessDenied
	}

	return s.buildProfileState(ctx, requesterID, profile)
}

func (s *profileService) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*ProfileState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	profile, err := s.repos.Profile.GetByUserID(ctx, input.RequesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, domain.ErrInvalidDisplayName
	}

	profile.DisplayName = displayName
	profile.Bio = nullableTrimmedStringWithLimit(input.Bio, 1000)
	profile.AvatarURL = normalizeOptionalURL(input.AvatarURL)
	profile.CoverURL = normalizeOptionalURL(input.CoverURL)
	profile.Location = nullableTrimmedStringWithLimit(input.Location, 100)
	profile.WebsiteURL = normalizeOptionalURL(input.WebsiteURL)
	profile.IsPublic = input.IsPublic

	if err := s.repos.Profile.Update(ctx, profile); err != nil {
		return nil, err
	}

	return s.getProfileByUserIDInternal(ctx, input.RequesterID, input.RequesterID)
}

func (s *profileService) UpdateProfileReadme(ctx context.Context, input UpdateProfileReadmeInput) (*ProfileState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	readme := nullableRawString(input.ReadmeMarkdown)

	if err := s.repos.Profile.UpdateReadme(ctx, input.RequesterID, readme); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	return s.getProfileByUserIDInternal(ctx, input.RequesterID, input.RequesterID)
}

func (s *profileService) ListProfileSocialLinks(ctx context.Context, requesterID, userID uuid.UUID) ([]*model.ProfileSocialLink, error) {
	profile, err := s.repos.Profile.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	if !canReadProfile(requesterID, profile) {
		return nil, domain.ErrProfileAccessDenied
	}

	links, err := s.repos.SocialLink.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if requesterID == userID {
		return links, nil
	}

	return filterVisibleSocialLinks(links), nil
}

func (s *profileService) UpsertProfileSocialLink(ctx context.Context, input UpsertProfileSocialLinkInput) (*model.ProfileSocialLink, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	if err := validateSocialPlatform(input.Platform); err != nil {
		return nil, err
	}

	urlValue, err := validateAndNormalizeURL(input.URL)
	if err != nil {
		return nil, err
	}

	label := nullableTrimmedStringWithLimit(input.Label, 64)

	if input.SocialLinkID == nil || *input.SocialLinkID == uuid.Nil {
		link := &model.ProfileSocialLink{
			UserID:    input.RequesterID,
			Platform:  input.Platform,
			URL:       urlValue,
			Label:     label,
			Position:  input.Position,
			IsVisible: input.IsVisible,
		}

		if err := s.repos.SocialLink.Create(ctx, link); err != nil {
			return nil, err
		}

		return s.repos.SocialLink.GetByID(ctx, link.ID)
	}

	link, err := s.repos.SocialLink.GetByID(ctx, *input.SocialLinkID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSocialLinkNotFound
		}
		return nil, err
	}

	if link.UserID != input.RequesterID {
		return nil, domain.ErrSocialLinkAccessDenied
	}

	link.Platform = input.Platform
	link.URL = urlValue
	link.Label = label
	link.Position = input.Position
	link.IsVisible = input.IsVisible

	if err := s.repos.SocialLink.Update(ctx, link); err != nil {
		return nil, err
	}

	return s.repos.SocialLink.GetByID(ctx, link.ID)
}

func (s *profileService) DeleteProfileSocialLink(ctx context.Context, requesterID, socialLinkID uuid.UUID) error {
	if requesterID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	link, err := s.repos.SocialLink.GetByID(ctx, socialLinkID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrSocialLinkNotFound
		}
		return err
	}

	if link.UserID != requesterID {
		return domain.ErrSocialLinkAccessDenied
	}

	return s.repos.SocialLink.DeleteByID(ctx, socialLinkID)
}

func (s *profileService) FollowUser(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	if followeeID == uuid.Nil {
		return domain.ErrProfileNotFound
	}
	if followerID == followeeID {
		return domain.ErrCannotFollowSelf
	}

	target, err := s.repos.Profile.GetByUserID(ctx, followeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProfileNotFound
		}
		return err
	}

	if !target.IsPublic {
		return domain.ErrProfileAccessDenied
	}

	return s.repos.Follow.Follow(ctx, &model.ProfileFollow{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
}

func (s *profileService) UnfollowUser(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	if followeeID == uuid.Nil {
		return domain.ErrProfileNotFound
	}
	if followerID == followeeID {
		return nil
	}

	return s.repos.Follow.Unfollow(ctx, followerID, followeeID)
}

func (s *profileService) ListFollowers(ctx context.Context, requesterID, userID uuid.UUID, pagination Pagination) ([]*ProfilePreview, int64, error) {
	profile, err := s.repos.Profile.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, domain.ErrProfileNotFound
		}
		return nil, 0, err
	}

	if !canReadProfile(requesterID, profile) {
		return nil, 0, domain.ErrProfileAccessDenied
	}

	items, total, err := s.repos.Follow.ListFollowers(ctx, userID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	return mapProfilePreviews(items), total, nil
}

func (s *profileService) ListFollowing(ctx context.Context, requesterID, userID uuid.UUID, pagination Pagination) ([]*ProfilePreview, int64, error) {
	profile, err := s.repos.Profile.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, domain.ErrProfileNotFound
		}
		return nil, 0, err
	}

	if !canReadProfile(requesterID, profile) {
		return nil, 0, domain.ErrProfileAccessDenied
	}

	items, total, err := s.repos.Follow.ListFollowing(ctx, userID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	return mapProfilePreviews(items), total, nil
}

func (s *profileService) ListAvailableTitles(ctx context.Context) ([]*model.ProfileTitle, error) {
	return s.repos.Title.ListActive(ctx)
}

func (s *profileService) SetProfileTitle(ctx context.Context, input SetProfileTitleInput) (*ProfileState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	titleCode := strings.TrimSpace(input.TitleCode)
	if titleCode == "" {
		return nil, domain.ErrProfileTitleNotFound
	}

	title, err := s.repos.Title.GetByCode(ctx, titleCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileTitleNotFound
		}
		return nil, err
	}

	if !title.IsActive {
		return nil, domain.ErrProfileTitleInactive
	}

	code := title.Code
	if err := s.repos.Profile.SetTitle(ctx, input.RequesterID, &code, model.ProfileTitleSourceManual); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	return s.getProfileByUserIDInternal(ctx, input.RequesterID, input.RequesterID)
}

func (s *profileService) getProfileByUserIDInternal(ctx context.Context, requesterID, userID uuid.UUID) (*ProfileState, error) {
	profile, err := s.repos.Profile.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	if !canReadProfile(requesterID, profile) {
		return nil, domain.ErrProfileAccessDenied
	}

	return s.buildProfileState(ctx, requesterID, profile)
}

func (s *profileService) buildProfileState(ctx context.Context, requesterID uuid.UUID, profile *model.Profile) (*ProfileState, error) {
	followersCount, err := s.repos.Follow.CountFollowers(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}

	followingCount, err := s.repos.Follow.CountFollowing(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}

	if requesterID != profile.UserID {
		profile.SocialLinks = filterVisibleSocialLinks(profile.SocialLinks)
	}

	return &ProfileState{
		Profile:        profile,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
	}, nil
}

func canReadProfile(requesterID uuid.UUID, profile *model.Profile) bool {
	if profile == nil {
		return false
	}

	if profile.IsPublic {
		return true
	}

	return requesterID != uuid.Nil && requesterID == profile.UserID
}

func normalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func nullableTrimmedStringWithLimit(v string, max int) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if max > 0 && len(v) > max {
		v = v[:max]
	}
	return &v
}

func nullableRawString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func normalizeOptionalURL(v string) *string {
	normalized, err := validateAndNormalizeURL(v)
	if err != nil {
		return nil
	}
	if normalized == "" {
		return nil
	}
	return &normalized
}

func validateAndNormalizeURL(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}

	parsed, err := url.ParseRequestURI(v)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", domain.ErrInvalidSocialURL
	}

	return parsed.String(), nil
}

func validateSocialPlatform(platform model.SocialPlatform) error {
	switch platform {
	case model.SocialPlatformTelegram,
		model.SocialPlatformGitHub,
		model.SocialPlatformVK,
		model.SocialPlatformLinkedIn,
		model.SocialPlatformX,
		model.SocialPlatformYouTube,
		model.SocialPlatformWebsite,
		model.SocialPlatformOther:
		return nil
	default:
		return domain.ErrInvalidSocialPlatform
	}
}

func filterVisibleSocialLinks(links []*model.ProfileSocialLink) []*model.ProfileSocialLink {
	if len(links) == 0 {
		return nil
	}

	result := make([]*model.ProfileSocialLink, 0, len(links))
	for _, link := range links {
		if link != nil && link.IsVisible {
			result = append(result, link)
		}
	}

	return result
}

func mapProfilePreviews(items []*repository.ProfilePreview) []*ProfilePreview {
	result := make([]*ProfilePreview, 0, len(items))
	for _, item := range items {
		result = append(result, &ProfilePreview{
			UserID:      item.UserID,
			Username:    item.Username,
			DisplayName: item.DisplayName,
			AvatarURL:   item.AvatarURL,
			TitleCode:   item.TitleCode,
			TitleLabel:  item.TitleLabel,
		})
	}
	return result
}
