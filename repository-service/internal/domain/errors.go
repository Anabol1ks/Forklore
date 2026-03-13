package domain

import "errors"

var (
	ErrRepositoryNotFound       = errors.New("repository not found")
	ErrRepositoryAccessDenied   = errors.New("repository access denied")
	ErrRepositorySlugTaken      = errors.New("repository slug already taken")
	ErrRepositoryCannotBeForked = errors.New("repository cannot be forked")

	ErrInvalidRepositoryVisibility = errors.New("invalid repository visibility")
	ErrInvalidRepositoryType       = errors.New("invalid repository type")

	ErrTagNotFound = errors.New("repository tag not found")
	ErrTagInactive = errors.New("repository tag is inactive")

	ErrUnauthorized = errors.New("unauthorized")
)
