package models

// ──────────────── Repository Enums ────────────────

type RepositoryVisibility string

const (
	RepositoryVisibilityPublic  RepositoryVisibility = "public"
	RepositoryVisibilityPrivate RepositoryVisibility = "private"
)

type RepositoryType string

const (
	RepositoryTypeArticle RepositoryType = "article"
	RepositoryTypeNotes   RepositoryType = "notes"
	RepositoryTypeMixed   RepositoryType = "mixed"
)

// ──────────────── Requests ────────────────

// CreateRepositoryRequest represents repository creation data.
type CreateRepositoryRequest struct {
	Name        string               `json:"name" binding:"required,min=3,max=100" example:"My Article"`
	Slug        string               `json:"slug" binding:"max=64" example:"my-article"`
	Description string               `json:"description" binding:"max=2000" example:"An interesting article"`
	TagID       string               `json:"tag_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Visibility  RepositoryVisibility `json:"visibility" binding:"required" enums:"public,private" example:"public"`
	Type        RepositoryType       `json:"type" binding:"required" enums:"article,notes,mixed" example:"article"`
}

// UpdateRepositoryRequest represents repository update data.
type UpdateRepositoryRequest struct {
	Name        string               `json:"name" binding:"omitempty,min=3,max=100" example:"Updated Article"`
	Slug        string               `json:"slug" binding:"omitempty,min=3,max=64" example:"updated-article"`
	Description string               `json:"description" binding:"max=2000" example:"Updated description"`
	TagID       string               `json:"tag_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Visibility  RepositoryVisibility `json:"visibility" enums:"public,private" example:"private"`
	Type        RepositoryType       `json:"type" enums:"article,notes,mixed" example:"notes"`
}

// ForkRepositoryRequest represents fork request.
type ForkRepositoryRequest struct {
	Name        string               `json:"name" binding:"max=100" example:"Forked Article"`
	Slug        string               `json:"slug" binding:"max=64" example:"forked-article"`
	Description string               `json:"description" binding:"max=2000" example:"My fork of the original"`
	Visibility  RepositoryVisibility `json:"visibility" binding:"required" enums:"public,private" example:"private"`
}

// ListRepositoriesRequest represents pagination for list operations.
type ListRepositoriesRequest struct {
	Limit  uint32 `form:"limit" binding:"gte=1,lte=100" example:"10"`
	Offset uint32 `form:"offset" example:"0"`
}

// ──────────────── Responses ────────────────

// RepositoryTagResponse represents a repository tag.
type RepositoryTagResponse struct {
	TagID       string `json:"tag_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" example:"Backend"`
	Slug        string `json:"slug" example:"backend"`
	Description string `json:"description" example:"Backend related repositories"`
	IsActive    bool   `json:"is_active" example:"true"`
	CreatedAt   string `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

// RepositoryResponse represents repository information.
type RepositoryResponse struct {
	RepoID        string                `json:"repo_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OwnerID       string                `json:"owner_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	OwnerUsername string                `json:"owner_username,omitempty" example:"anabol1ks"`
	Name          string                `json:"name" example:"My Article"`
	Slug          string                `json:"slug" example:"my-article"`
	Description   *string               `json:"description,omitempty" example:"An interesting article"`
	Visibility    string                `json:"visibility" example:"public" enums:"public,private"`
	Type          string                `json:"type" example:"article" enums:"article,notes,mixed"`
	Tag           RepositoryTagResponse `json:"tag"`
	ParentRepoID  *string               `json:"parent_repo_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	CreatedAt     string                `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt     string                `json:"updated_at" example:"2026-01-01T00:00:00Z"`
	DeletedAt     *string               `json:"deleted_at,omitempty" example:"2026-01-02T00:00:00Z"`
}

// CreateRepositoryResponse represents the response after creating a repository.
type CreateRepositoryResponse struct {
	Repository RepositoryResponse `json:"repository"`
}

// UpdateRepositoryResponse represents the response after updating a repository.
type UpdateRepositoryResponse struct {
	Repository RepositoryResponse `json:"repository"`
}

// ForkRepositoryResponse represents the response after forking a repository.
type ForkRepositoryResponse struct {
	Repository RepositoryResponse `json:"repository"`
}

// ListRepositoriesResponse represents paginated list of repositories.
type ListRepositoriesResponse struct {
	Repositories []RepositoryResponse `json:"repositories"`
	Total        uint64               `json:"total" example:"42"`
}

// ListRepositoryTagsResponse represents list of available repository tags.
type ListRepositoryTagsResponse struct {
	Tags []RepositoryTagResponse `json:"tags"`
}

// GetRepositoryResponse represents response for getting a single repository.
type GetRepositoryResponse struct {
	Repository RepositoryResponse `json:"repository"`
}

// RepositoryStarStateResponse represents star state for a repository.
type RepositoryStarStateResponse struct {
	RepoID     string `json:"repo_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Starred    bool   `json:"starred" example:"true"`
	StarsCount uint64 `json:"stars_count" example:"12"`
}
