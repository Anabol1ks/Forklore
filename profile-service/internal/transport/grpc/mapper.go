package grpcserver

import (
	"profile-service/internal/model"
	"profile-service/internal/service"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
)

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{Value: id.String()}
}

func parseProtoUUID(id *commonv1.UUID, fieldName string) (uuid.UUID, error) {
	if id == nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	value := strings.TrimSpace(id.GetValue())
	if value == "" {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s must be a valid uuid", fieldName)
	}

	return parsed, nil
}

func parseOptionalProtoUUID(id *commonv1.UUID, fieldName string) (*uuid.UUID, error) {
	if id == nil || strings.TrimSpace(id.GetValue()) == "" {
		return nil, nil
	}

	parsed, err := parseProtoUUID(id, fieldName)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoProfileTitleSource(src model.ProfileTitleSource) profilev1.ProfileTitleSource {
	switch src {
	case model.ProfileTitleSourceSystem:
		return profilev1.ProfileTitleSource_PROFILE_TITLE_SOURCE_SYSTEM
	case model.ProfileTitleSourceManual:
		return profilev1.ProfileTitleSource_PROFILE_TITLE_SOURCE_MANUAL
	case model.ProfileTitleSourceAchievement:
		return profilev1.ProfileTitleSource_PROFILE_TITLE_SOURCE_ACHIEVEMENT
	default:
		return profilev1.ProfileTitleSource_PROFILE_TITLE_SOURCE_UNSPECIFIED
	}
}

func toProtoSocialPlatform(platform model.SocialPlatform) profilev1.SocialPlatform {
	switch platform {
	case model.SocialPlatformTelegram:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_TELEGRAM
	case model.SocialPlatformGitHub:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_GITHUB
	case model.SocialPlatformVK:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_VK
	case model.SocialPlatformLinkedIn:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_LINKEDIN
	case model.SocialPlatformX:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_X
	case model.SocialPlatformYouTube:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_YOUTUBE
	case model.SocialPlatformWebsite:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_WEBSITE
	case model.SocialPlatformOther:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_OTHER
	default:
		return profilev1.SocialPlatform_SOCIAL_PLATFORM_UNSPECIFIED
	}
}

func toModelSocialPlatform(platform profilev1.SocialPlatform) model.SocialPlatform {
	switch platform {
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_TELEGRAM:
		return model.SocialPlatformTelegram
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_GITHUB:
		return model.SocialPlatformGitHub
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_VK:
		return model.SocialPlatformVK
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_LINKEDIN:
		return model.SocialPlatformLinkedIn
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_X:
		return model.SocialPlatformX
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_YOUTUBE:
		return model.SocialPlatformYouTube
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_WEBSITE:
		return model.SocialPlatformWebsite
	case profilev1.SocialPlatform_SOCIAL_PLATFORM_OTHER:
		return model.SocialPlatformOther
	default:
		return ""
	}
}

func toProtoProfileTitle(title *model.ProfileTitle) *profilev1.ProfileTitle {
	if title == nil {
		return nil
	}

	return &profilev1.ProfileTitle{
		Code:        title.Code,
		Label:       title.Label,
		Description: derefString(title.Description),
		SortOrder:   title.SortOrder,
		IsActive:    title.IsActive,
		IsSystem:    title.IsSystem,
	}
}

func toProtoProfileSocialLink(link *model.ProfileSocialLink) *profilev1.ProfileSocialLink {
	if link == nil {
		return nil
	}

	return &profilev1.ProfileSocialLink{
		SocialLinkId: toProtoUUID(link.ID),
		UserId:       toProtoUUID(link.UserID),
		Platform:     toProtoSocialPlatform(link.Platform),
		Url:          link.URL,
		Label:        derefString(link.Label),
		Position:     link.Position,
		IsVisible:    link.IsVisible,
		CreatedAt:    toProtoTimestamp(link.CreatedAt),
		UpdatedAt:    toProtoTimestamp(link.UpdatedAt),
	}
}

func toProtoProfileSocialLinks(items []*model.ProfileSocialLink) []*profilev1.ProfileSocialLink {
	result := make([]*profilev1.ProfileSocialLink, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoProfileSocialLink(item))
	}
	return result
}

func toProtoProfile(profile *model.Profile, followersCount, followingCount int64) *profilev1.Profile {
	if profile == nil {
		return nil
	}

	return &profilev1.Profile{
		UserId:         toProtoUUID(profile.UserID),
		Username:       profile.Username,
		DisplayName:    profile.DisplayName,
		Bio:            derefString(profile.Bio),
		AvatarUrl:      derefString(profile.AvatarURL),
		CoverUrl:       derefString(profile.CoverURL),
		Location:       derefString(profile.Location),
		WebsiteUrl:     derefString(profile.WebsiteURL),
		ReadmeMarkdown: derefString(profile.ReadmeMarkdown),
		IsPublic:       profile.IsPublic,
		Title:          toProtoProfileTitle(profile.Title),
		TitleSource:    toProtoProfileTitleSource(profile.TitleSource),
		FollowersCount: uint64(followersCount),
		FollowingCount: uint64(followingCount),
		SocialLinks:    toProtoProfileSocialLinks(profile.SocialLinks),
		CreatedAt:      toProtoTimestamp(profile.CreatedAt),
		UpdatedAt:      toProtoTimestamp(profile.UpdatedAt),
	}
}

func toProtoProfilePreview(item *service.ProfilePreview) *profilev1.ProfilePreview {
	if item == nil {
		return nil
	}

	var title *profilev1.ProfileTitle
	if item.TitleCode != nil || item.TitleLabel != nil {
		title = &profilev1.ProfileTitle{
			Code:  derefString(item.TitleCode),
			Label: derefString(item.TitleLabel),
		}
	}

	return &profilev1.ProfilePreview{
		UserId:      toProtoUUID(item.UserID),
		Username:    item.Username,
		DisplayName: item.DisplayName,
		AvatarUrl:   derefString(item.AvatarURL),
		Title:       title,
	}
}

func toProtoProfilePreviews(items []*service.ProfilePreview) []*profilev1.ProfilePreview {
	result := make([]*profilev1.ProfilePreview, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoProfilePreview(item))
	}
	return result
}

func toProtoProfileTitles(items []*model.ProfileTitle) []*profilev1.ProfileTitle {
	result := make([]*profilev1.ProfileTitle, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoProfileTitle(item))
	}
	return result
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
