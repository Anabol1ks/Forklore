package models

type SocialPlatform string

const (
	SocialPlatformTelegram SocialPlatform = "telegram"
	SocialPlatformGithub   SocialPlatform = "github"
	SocialPlatformVK       SocialPlatform = "vk"
	SocialPlatformLinkedIn SocialPlatform = "linkedin"
	SocialPlatformX        SocialPlatform = "x"
	SocialPlatformYoutube  SocialPlatform = "youtube"
	SocialPlatformWebsite  SocialPlatform = "website"
	SocialPlatformOther    SocialPlatform = "other"
)

type ProfileTitleSource string

const (
	ProfileTitleSourceSystem      ProfileTitleSource = "system"
	ProfileTitleSourceManual      ProfileTitleSource = "manual"
	ProfileTitleSourceAchievement ProfileTitleSource = "achievement"
)

type ProfileTitleResponse struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	SortOrder   int32  `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
	IsSystem    bool   `json:"is_system"`
}

type ProfileSocialLinkResponse struct {
	SocialLinkID string `json:"social_link_id"`
	UserID       string `json:"user_id"`
	Platform     string `json:"platform"`
	URL          string `json:"url"`
	Label        string `json:"label,omitempty"`
	Position     int32  `json:"position"`
	IsVisible    bool   `json:"is_visible"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type ProfileResponse struct {
	UserID         string                      `json:"user_id"`
	Username       string                      `json:"username"`
	DisplayName    string                      `json:"display_name"`
	Bio            string                      `json:"bio,omitempty"`
	AvatarURL      string                      `json:"avatar_url,omitempty"`
	CoverURL       string                      `json:"cover_url,omitempty"`
	Location       string                      `json:"location,omitempty"`
	WebsiteURL     string                      `json:"website_url,omitempty"`
	ReadmeMarkdown string                      `json:"readme_markdown,omitempty"`
	IsPublic       bool                        `json:"is_public"`
	Title          *ProfileTitleResponse       `json:"title,omitempty"`
	TitleSource    string                      `json:"title_source"`
	FollowersCount uint64                      `json:"followers_count"`
	FollowingCount uint64                      `json:"following_count"`
	SocialLinks    []ProfileSocialLinkResponse `json:"social_links"`
	CreatedAt      string                      `json:"created_at"`
	UpdatedAt      string                      `json:"updated_at,omitempty"`
}

type ProfilePreviewResponse struct {
	UserID      string                `json:"user_id"`
	Username    string                `json:"username"`
	DisplayName string                `json:"display_name"`
	AvatarURL   string                `json:"avatar_url,omitempty"`
	Title       *ProfileTitleResponse `json:"title,omitempty"`
}

type GetProfileResponse struct {
	Profile ProfileResponse `json:"profile"`
}

type UploadProfileImageResponse struct {
	Profile ProfileResponse `json:"profile"`
	Field   string          `json:"field"`
	URL     string          `json:"url"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" binding:"required,max=100"`
	Bio         string `json:"bio" binding:"max=1000"`
	AvatarURL   string `json:"avatar_url" binding:"max=2048"`
	CoverURL    string `json:"cover_url" binding:"max=2048"`
	Location    string `json:"location" binding:"max=100"`
	WebsiteURL  string `json:"website_url" binding:"max=2048"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateProfileReadmeRequest struct {
	ReadmeMarkdown string `json:"readme_markdown"`
}

type UpsertProfileSocialLinkRequest struct {
	SocialLinkID string `json:"social_link_id,omitempty"`
	Platform     string `json:"platform" binding:"required"`
	URL          string `json:"url" binding:"required,max=2048"`
	Label        string `json:"label" binding:"max=64"`
	Position     int32  `json:"position"`
	IsVisible    bool   `json:"is_visible"`
}

type ListProfileSocialLinksResponse struct {
	SocialLinks []ProfileSocialLinkResponse `json:"social_links"`
}

type UpsertProfileSocialLinkResponse struct {
	SocialLink ProfileSocialLinkResponse `json:"social_link"`
}

type SetProfileTitleRequest struct {
	TitleCode string `json:"title_code" binding:"required,max=64"`
}

type ListProfilePreviewsResponse struct {
	Profiles []ProfilePreviewResponse `json:"profiles"`
	Total    uint64                   `json:"total"`
}

type ListAvailableTitlesResponse struct {
	Titles []ProfileTitleResponse `json:"titles"`
}
