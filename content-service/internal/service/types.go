package service

import (
	"content-service/internal/model"
	"context"

	"github.com/google/uuid"
)

type Pagination struct {
	Limit  int
	Offset int
}

type RepositoryAccessChecker interface {
	EnsureCanRead(ctx context.Context, repoID, requesterID uuid.UUID) error
	EnsureCanWrite(ctx context.Context, repoID, requesterID uuid.UUID) error
	GetRepositoryMetadata(ctx context.Context, repoID uuid.UUID) (*RepositoryMetadata, error)
}

type DocumentState struct {
	Document       *model.Document
	Draft          *model.DocumentDraft
	CurrentVersion *model.DocumentVersion
}

type DocumentVersionResult struct {
	Document *model.Document
	Version  *model.DocumentVersion
}

type FileState struct {
	File           *model.File
	CurrentVersion *model.FileVersion
}

type FileVersionResult struct {
	File    *model.File
	Version *model.FileVersion
}

type CreateDocumentInput struct {
	RequesterID    uuid.UUID
	RepoID         uuid.UUID
	Title          string
	Slug           string
	Format         model.DocumentFormat
	InitialContent string
	ChangeSummary  string
}

type SaveDocumentDraftInput struct {
	RequesterID uuid.UUID
	DocumentID  uuid.UUID
	Content     string
}

type CreateDocumentVersionInput struct {
	RequesterID   uuid.UUID
	DocumentID    uuid.UUID
	Content       string
	ChangeSummary string
}

type RestoreDocumentVersionInput struct {
	RequesterID   uuid.UUID
	DocumentID    uuid.UUID
	VersionID     uuid.UUID
	ChangeSummary string
}

type CreateFileInput struct {
	RequesterID    uuid.UUID
	RepoID         uuid.UUID
	FileName       string
	StorageKey     string
	MimeType       string
	SizeBytes      uint64
	ChecksumSHA256 string
	ChangeSummary  string
}

type AddFileVersionInput struct {
	RequesterID    uuid.UUID
	FileID         uuid.UUID
	StorageKey     string
	MimeType       string
	SizeBytes      uint64
	ChecksumSHA256 string
	ChangeSummary  string
}

type RestoreFileVersionInput struct {
	RequesterID   uuid.UUID
	FileID        uuid.UUID
	VersionID     uuid.UUID
	ChangeSummary string
}

type ContentService interface {
	CreateDocument(ctx context.Context, input CreateDocumentInput) (*DocumentState, error)
	GetDocumentByID(ctx context.Context, requesterID, documentID uuid.UUID) (*DocumentState, error)
	ListRepositoryDocuments(ctx context.Context, requesterID, repoID uuid.UUID, pagination Pagination) ([]*model.Document, int64, error)
	SaveDocumentDraft(ctx context.Context, input SaveDocumentDraftInput) (*DocumentState, error)
	CreateDocumentVersion(ctx context.Context, input CreateDocumentVersionInput) (*DocumentVersionResult, error)
	GetDocumentVersionByID(ctx context.Context, requesterID, versionID uuid.UUID) (*model.DocumentVersion, error)
	ListDocumentVersions(ctx context.Context, requesterID, documentID uuid.UUID, pagination Pagination) ([]*model.DocumentVersion, int64, error)
	RestoreDocumentVersion(ctx context.Context, input RestoreDocumentVersionInput) (*DocumentVersionResult, error)
	DeleteDocument(ctx context.Context, requesterID, documentID uuid.UUID) error

	CreateFile(ctx context.Context, input CreateFileInput) (*FileState, error)
	GetFileByID(ctx context.Context, requesterID, fileID uuid.UUID) (*FileState, error)
	ListRepositoryFiles(ctx context.Context, requesterID, repoID uuid.UUID, pagination Pagination) ([]*model.File, int64, error)
	AddFileVersion(ctx context.Context, input AddFileVersionInput) (*FileVersionResult, error)
	GetFileVersionByID(ctx context.Context, requesterID, versionID uuid.UUID) (*model.FileVersion, error)
	ListFileVersions(ctx context.Context, requesterID, fileID uuid.UUID, pagination Pagination) ([]*model.FileVersion, int64, error)
	RestoreFileVersion(ctx context.Context, input RestoreFileVersionInput) (*FileVersionResult, error)
	DeleteFile(ctx context.Context, requesterID, fileID uuid.UUID) error
}
