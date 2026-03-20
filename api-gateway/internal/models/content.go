package models

// ──── Document Models ────

type CreateDocumentRequest struct {
	Title          string `json:"title" binding:"required,min=1,max=200"`
	Slug           string `json:"slug" binding:"max=100"`
	InitialContent string `json:"initial_content"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type SaveDraftRequest struct {
	Content string `json:"content"`
}

type CreateVersionRequest struct {
	Content       string `json:"content"`
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

type RestoreVersionRequest struct {
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

// DocumentDetailResponse represents document information.
type DocumentDetailResponse struct {
	DocumentID           string                 `json:"document_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RepoID               string                 `json:"repo_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	AuthorID             string                 `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Title                string                 `json:"title" example:"My Document"`
	Slug                 string                 `json:"slug" example:"my-document"`
	Format               string                 `json:"format" example:"markdown"`
	CurrentVersionID     string                 `json:"current_version_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	LatestDraftUpdatedAt *string                `json:"latest_draft_updated_at,omitempty" example:"2026-01-01T00:00:00Z"`
	CreatedAt            string                 `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt            *string                `json:"updated_at,omitempty" example:"2026-01-02T00:00:00Z"`
	DeletedAt            *string                `json:"deleted_at,omitempty" example:"2026-01-03T00:00:00Z"`
	Draft                *DocumentDraftResponse `json:"draft,omitempty"`
	CurrentVersion       *DocumentVersionDetail `json:"current_version,omitempty"`
}

type DocumentDraftResponse struct {
	DocumentID string `json:"document_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content    string `json:"content"`
	UpdatedBy  string `json:"updated_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	UpdatedAt  string `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type DocumentResponse struct {
	Document DocumentDetailResponse `json:"document"`
}

type DocumentListResponse struct {
	Documents []DocumentDetailResponse `json:"documents"`
	Total     uint64                   `json:"total" example:"42"`
}

// ──── Document Version Models ────

type DocumentVersionDetail struct {
	VersionID     string `json:"version_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	DocumentID    string `json:"document_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID      string `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	VersionNumber uint32 `json:"version_number" example:"1"`
	Content       string `json:"content"`
	ChangeSummary string `json:"change_summary" example:"Initial version"`
	CreatedAt     string `json:"created_at" example:"2026-01-01T00:00:00Z"`
}

type VersionResponse struct {
	Version DocumentVersionDetail `json:"version"`
}

type VersionListResponse struct {
	Versions []DocumentVersionDetail `json:"versions"`
	Total    uint64                  `json:"total" example:"42"`
}

// ──── File Models ────

type CreateFileRequest struct {
	FileName       string `json:"file_name" binding:"required,min=1,max=255"`
	StorageKey     string `json:"storage_key" binding:"required,min=1,max=2048"`
	MimeType       string `json:"mime_type" binding:"required,min=1,max=255"`
	SizeBytes      uint64 `json:"size_bytes" binding:"required"`
	ChecksumSHA256 string `json:"checksum_sha256" binding:"max=64"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type AddFileVersionRequest struct {
	StorageKey     string `json:"storage_key" binding:"required,min=1,max=2048"`
	MimeType       string `json:"mime_type" binding:"required,min=1,max=255"`
	SizeBytes      uint64 `json:"size_bytes" binding:"required"`
	ChecksumSHA256 string `json:"checksum_sha256" binding:"max=64"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type RestoreFileVersionRequest struct {
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

// FileDetailResponse represents file information.
type FileDetailResponse struct {
	FileID           string  `json:"file_id" example:"550e8400-e29b-41d4-a716-446655440004"`
	RepoID           string  `json:"repo_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	UploadedBy       string  `json:"uploaded_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	FileName         string  `json:"file_name" example:"document.pdf"`
	CurrentVersionID string  `json:"current_version_id" example:"550e8400-e29b-41d4-a716-446655440005"`
	CreatedAt        string  `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt        *string `json:"updated_at,omitempty" example:"2026-01-02T00:00:00Z"`
	DeletedAt        *string `json:"deleted_at,omitempty" example:"2026-01-03T00:00:00Z"`
}

type FileResponse struct {
	File FileDetailResponse `json:"file"`
}

type FileListResponse struct {
	Files []FileDetailResponse `json:"files"`
	Total uint64               `json:"total" example:"42"`
}

// ──── File Version Models ────

type FileVersionDetail struct {
	VersionID      string `json:"version_id" example:"550e8400-e29b-41d4-a716-446655440005"`
	FileID         string `json:"file_id" example:"550e8400-e29b-41d4-a716-446655440004"`
	UploadedBy     string `json:"uploaded_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	VersionNumber  uint32 `json:"version_number" example:"1"`
	StorageKey     string `json:"storage_key" example:"uploads/file-123.pdf"`
	MimeType       string `json:"mime_type" example:"application/pdf"`
	SizeBytes      uint64 `json:"size_bytes" example:"1024000"`
	ChecksumSHA256 string `json:"checksum_sha256" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	ChangeSummary  string `json:"change_summary" example:"Updated file"`
	CreatedAt      string `json:"created_at" example:"2026-01-01T00:00:00Z"`
}

type FileVersionResponse struct {
	Version FileVersionDetail `json:"version"`
}

type FileVersionListResponse struct {
	Versions []FileVersionDetail `json:"versions"`
	Total    uint64              `json:"total" example:"42"`
}
