package domain

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized")

	ErrProfileNotFound     = errors.New("profile not found")
	ErrProfileAccessDenied = errors.New("profile access denied")

	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrInvalidUsername    = errors.New("invalid username")

	ErrProfileTitleNotFound = errors.New("profile title not found")
	ErrProfileTitleInactive = errors.New("profile title is inactive")

	ErrSocialLinkNotFound     = errors.New("social link not found")
	ErrSocialLinkAccessDenied = errors.New("social link access denied")
	ErrInvalidSocialPlatform  = errors.New("invalid social platform")
	ErrInvalidSocialURL       = errors.New("invalid social url")

	ErrCannotFollowSelf = errors.New("cannot follow yourself")
)
