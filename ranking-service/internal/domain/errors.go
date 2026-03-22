package domain

import "errors"

var (
	ErrInvalidLimit = errors.New("invalid limit")
	ErrInvalidTagID = errors.New("invalid tag id")
)
