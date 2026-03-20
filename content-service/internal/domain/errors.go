package domain

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized")

	ErrRepositoryNotFound  = errors.New("repository not found")
	ErrContentAccessDenied = errors.New("content access denied")

	ErrDocumentNotFound        = errors.New("document not found")
	ErrDocumentVersionNotFound = errors.New("document version not found")
	ErrDocumentSlugTaken       = errors.New("document slug already taken")
	ErrInvalidDocumentFormat   = errors.New("invalid document format")
	ErrInvalidDocumentTitle    = errors.New("invalid document title")
	ErrInvalidDocumentSlug     = errors.New("invalid document slug")

	ErrFileNotFound        = errors.New("file not found")
	ErrFileVersionNotFound = errors.New("file version not found")
	ErrInvalidFileName     = errors.New("invalid file name")
)
