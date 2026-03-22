package models

import "time"

type SearchRequest struct {
	Query       string   `json:"query" binding:"omitempty,max=500"`
	EntityTypes []string `json:"entity_types"`
	TagID       string   `json:"tag_id,omitempty"`
	OwnerID     string   `json:"owner_id,omitempty"`
	RepoID      string   `json:"repo_id,omitempty"`
	Limit       uint32   `json:"limit" binding:"omitempty,gte=1,lte=100"`
	Offset      uint32   `json:"offset" binding:"omitempty,gte=0"`
}

type SearchHitResponse struct {
	EntityType  string  `json:"entity_type" example:"repository"`
	EntityID    string  `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RepoID      *string `json:"repo_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	OwnerID     *string `json:"owner_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	TagID       *string `json:"tag_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Snippet     *string `json:"snippet,omitempty"`
	Rank        float64 `json:"rank" example:"0.93"`
	UpdatedAt   string  `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type SearchResponse struct {
	Hits  []SearchHitResponse `json:"hits"`
	Total uint64              `json:"total" example:"42"`
}

type SearchUsersRequest struct {
	Query      string `json:"query" binding:"omitempty,max=500"`
	University string `json:"university,omitempty" binding:"omitempty,max=100"`
	TagID      string `json:"tag_id,omitempty"`
	Limit      uint32 `json:"limit" binding:"omitempty,gte=1,lte=100"`
	Offset     uint32 `json:"offset" binding:"omitempty,gte=0"`
}

type SearchUserHitResponse struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	DisplayName       string `json:"display_name,omitempty"`
	AvatarURL         string `json:"avatar_url,omitempty"`
	University        string `json:"university,omitempty"`
	RepositoriesCount uint32 `json:"repositories_count"`
}

type SearchUsersResponse struct {
	Users []SearchUserHitResponse `json:"users"`
	Total uint64                  `json:"total"`
}

type UpsertRepositoryIndexRequest struct {
	RepoID      string    `json:"repo_id" binding:"required"`
	OwnerID     string    `json:"owner_id" binding:"required"`
	TagID       string    `json:"tag_id" binding:"required"`
	Title       string    `json:"title" binding:"required,min=1,max=255"`
	Description string    `json:"description" binding:"max=4000"`
	TagName     string    `json:"tag_name" binding:"max=128"`
	IsPublic    bool      `json:"is_public"`
	UpdatedAt   time.Time `json:"updated_at" binding:"required"`
}

type UpsertDocumentIndexRequest struct {
	DocumentID string    `json:"document_id" binding:"required"`
	RepoID     string    `json:"repo_id" binding:"required"`
	OwnerID    string    `json:"owner_id" binding:"required"`
	TagID      string    `json:"tag_id" binding:"required"`
	Title      string    `json:"title" binding:"required,min=1,max=255"`
	Content    string    `json:"content"`
	TagName    string    `json:"tag_name" binding:"max=128"`
	IsPublic   bool      `json:"is_public"`
	UpdatedAt  time.Time `json:"updated_at" binding:"required"`
}

type UpsertFileIndexRequest struct {
	FileID    string    `json:"file_id" binding:"required"`
	RepoID    string    `json:"repo_id" binding:"required"`
	OwnerID   string    `json:"owner_id" binding:"required"`
	TagID     string    `json:"tag_id" binding:"required"`
	FileName  string    `json:"file_name" binding:"required,min=1,max=255"`
	MimeType  string    `json:"mime_type" binding:"max=255"`
	TagName   string    `json:"tag_name" binding:"max=128"`
	IsPublic  bool      `json:"is_public"`
	UpdatedAt time.Time `json:"updated_at" binding:"required"`
}
