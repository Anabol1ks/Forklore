package repository

import (
	"context"

	"content-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListParams struct {
	Limit  int
	Offset int
}

type DocumentRepository interface {
	Create(ctx context.Context, document *model.Document) error
	GetByID(ctx context.Context, documentID uuid.UUID) (*model.Document, error)
	GetByIDForUpdate(ctx context.Context, documentID uuid.UUID) (*model.Document, error)
	GetByRepoAndSlug(ctx context.Context, repoID uuid.UUID, slug string) (*model.Document, error)
	Update(ctx context.Context, document *model.Document) error
	DeleteByID(ctx context.Context, documentID uuid.UUID) error
	ListByRepoID(ctx context.Context, repoID uuid.UUID, params ListParams) ([]*model.Document, int64, error)
}

type DocumentDraftRepository interface {
	Create(ctx context.Context, draft *model.DocumentDraft) error
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) (*model.DocumentDraft, error)
	Upsert(ctx context.Context, draft *model.DocumentDraft) error
	DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
}

type DocumentVersionRepository interface {
	Create(ctx context.Context, version *model.DocumentVersion) error
	GetByID(ctx context.Context, versionID uuid.UUID) (*model.DocumentVersion, error)
	GetByDocumentAndVersionID(ctx context.Context, documentID, versionID uuid.UUID) (*model.DocumentVersion, error)
	GetLatestVersionNumber(ctx context.Context, documentID uuid.UUID) (uint32, error)
	ListByDocumentID(ctx context.Context, documentID uuid.UUID, params ListParams) ([]*model.DocumentVersion, int64, error)
}

type FileRepository interface {
	Create(ctx context.Context, file *model.File) error
	GetByID(ctx context.Context, fileID uuid.UUID) (*model.File, error)
	GetByIDForUpdate(ctx context.Context, fileID uuid.UUID) (*model.File, error)
	Update(ctx context.Context, file *model.File) error
	DeleteByID(ctx context.Context, fileID uuid.UUID) error
	ListByRepoID(ctx context.Context, repoID uuid.UUID, params ListParams) ([]*model.File, int64, error)
}

type FileVersionRepository interface {
	Create(ctx context.Context, version *model.FileVersion) error
	GetByID(ctx context.Context, versionID uuid.UUID) (*model.FileVersion, error)
	GetByFileAndVersionID(ctx context.Context, fileID, versionID uuid.UUID) (*model.FileVersion, error)
	GetLatestVersionNumber(ctx context.Context, fileID uuid.UUID) (uint32, error)
	ListByFileID(ctx context.Context, fileID uuid.UUID, params ListParams) ([]*model.FileVersion, int64, error)
}

type Repository struct {
	db *gorm.DB

	Document        DocumentRepository
	DocumentDraft   DocumentDraftRepository
	DocumentVersion DocumentVersionRepository
	File            FileRepository
	FileVersion     FileVersionRepository
}
