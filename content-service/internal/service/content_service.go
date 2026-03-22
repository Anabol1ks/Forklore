package service

import (
	"content-service/internal/domain"
	"content-service/internal/model"
	"content-service/internal/repository"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentService struct {
	repos      *repository.Repository
	repoAccess RepositoryAccessChecker
	publisher  SearchEventPublisher
	logger     *zap.Logger
}

func NewContentService(
	repos *repository.Repository,
	repoAccess RepositoryAccessChecker,
) ContentService {
	return NewContentServiceWithPublisher(repos, repoAccess, nil, nil)
}

func NewContentServiceWithPublisher(
	repos *repository.Repository,
	repoAccess RepositoryAccessChecker,
	publisher SearchEventPublisher,
	logger *zap.Logger,
) ContentService {
	if publisher == nil {
		publisher = NewNoopSearchEventPublisher()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &contentService{
		repos:      repos,
		repoAccess: repoAccess,
		publisher:  publisher,
		logger:     logger,
	}
}

func (s *contentService) CreateDocument(ctx context.Context, input CreateDocumentInput) (*DocumentState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}
	if input.RepoID == uuid.Nil {
		return nil, domain.ErrRepositoryNotFound
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, input.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, domain.ErrInvalidDocumentTitle
	}

	format := normalizeDocumentFormat(input.Format)
	if err := validateDocumentFormat(format); err != nil {
		return nil, err
	}

	slug := normalizeDocumentSlug(input.Slug)
	if slug == "" {
		slug = slugifyDocument(title)
	}
	if slug == "" {
		slug = "document"
	}

	userProvidedSlug := strings.TrimSpace(input.Slug) != ""
	if userProvidedSlug {
		if _, err := s.repos.Document.GetByRepoAndSlug(ctx, input.RepoID, slug); err == nil {
			return nil, domain.ErrDocumentSlugTaken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		resolvedSlug, err := s.findAvailableDocumentSlug(ctx, input.RepoID, slug)
		if err != nil {
			return nil, err
		}
		slug = resolvedSlug
	}

	now := time.Now().UTC()
	initialContent := input.InitialContent
	changeSummary := nullableTrimmedString(input.ChangeSummary)

	var createdDocumentID uuid.UUID

	err := s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		document := &model.Document{
			RepoID:    input.RepoID,
			AuthorID:  input.RequesterID,
			Title:     title,
			Slug:      slug,
			Format:    format,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := txRepo.Document.Create(ctx, document); err != nil {
			if mapped := mapDocumentPersistenceError(err); mapped != nil {
				return mapped
			}
			return err
		}

		draft := &model.DocumentDraft{
			DocumentID: document.ID,
			Content:    initialContent,
			UpdatedBy:  input.RequesterID,
			UpdatedAt:  now,
		}
		if err := txRepo.DocumentDraft.Create(ctx, draft); err != nil {
			return err
		}

		version := &model.DocumentVersion{
			DocumentID:    document.ID,
			AuthorID:      input.RequesterID,
			VersionNumber: 1,
			Content:       initialContent,
			ChangeSummary: changeSummary,
			CreatedAt:     now,
		}
		if err := txRepo.DocumentVersion.Create(ctx, version); err != nil {
			return err
		}

		document.CurrentVersionID = &version.ID
		document.LatestDraftUpdatedAt = &now
		document.UpdatedAt = now

		if err := txRepo.Document.Update(ctx, document); err != nil {
			return err
		}

		createdDocumentID = document.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	state, err := s.GetDocumentByID(ctx, input.RequesterID, createdDocumentID)
	if err != nil {
		return nil, err
	}

	s.publishDocumentUpserted(ctx, state.Document, state.CurrentVersion)

	return state, nil
}

func (s *contentService) GetDocumentByID(ctx context.Context, requesterID, documentID uuid.UUID) (*DocumentState, error) {
	document, err := s.repos.Document.GetByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, document.RepoID, requesterID); err != nil {
		return nil, err
	}

	return &DocumentState{
		Document:       document,
		Draft:          document.Draft,
		CurrentVersion: document.CurrentVersion,
	}, nil
}

func (s *contentService) ListRepositoryDocuments(ctx context.Context, requesterID, repoID uuid.UUID, pagination Pagination) ([]*model.Document, int64, error) {
	if repoID == uuid.Nil {
		return nil, 0, domain.ErrRepositoryNotFound
	}

	if err := s.repoAccess.EnsureCanRead(ctx, repoID, requesterID); err != nil {
		return nil, 0, err
	}

	return s.repos.Document.ListByRepoID(ctx, repoID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (s *contentService) SaveDocumentDraft(ctx context.Context, input SaveDocumentDraftInput) (*DocumentState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	document, err := s.repos.Document.GetByID(ctx, input.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, document.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		draft := &model.DocumentDraft{
			DocumentID: document.ID,
			Content:    input.Content,
			UpdatedBy:  input.RequesterID,
			UpdatedAt:  now,
		}
		if err := txRepo.DocumentDraft.Upsert(ctx, draft); err != nil {
			return err
		}

		lockedDocument, err := txRepo.Document.GetByIDForUpdate(ctx, document.ID)
		if err != nil {
			return err
		}

		lockedDocument.LatestDraftUpdatedAt = &now
		lockedDocument.UpdatedAt = now

		return txRepo.Document.Update(ctx, lockedDocument)
	})
	if err != nil {
		return nil, err
	}

	return s.GetDocumentByID(ctx, input.RequesterID, input.DocumentID)
}

func (s *contentService) CreateDocumentVersion(ctx context.Context, input CreateDocumentVersionInput) (*DocumentVersionResult, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	document, err := s.repos.Document.GetByID(ctx, input.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, document.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	changeSummary := nullableTrimmedString(input.ChangeSummary)

	var createdVersionID uuid.UUID

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		lockedDocument, err := txRepo.Document.GetByIDForUpdate(ctx, document.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrDocumentNotFound
			}
			return err
		}

		latestNumber, err := txRepo.DocumentVersion.GetLatestVersionNumber(ctx, lockedDocument.ID)
		if err != nil {
			return err
		}

		version := &model.DocumentVersion{
			DocumentID:    lockedDocument.ID,
			AuthorID:      input.RequesterID,
			VersionNumber: latestNumber + 1,
			Content:       input.Content,
			ChangeSummary: changeSummary,
			CreatedAt:     now,
		}
		if err := txRepo.DocumentVersion.Create(ctx, version); err != nil {
			return err
		}

		draft := &model.DocumentDraft{
			DocumentID: lockedDocument.ID,
			Content:    input.Content,
			UpdatedBy:  input.RequesterID,
			UpdatedAt:  now,
		}
		if err := txRepo.DocumentDraft.Upsert(ctx, draft); err != nil {
			return err
		}

		lockedDocument.CurrentVersionID = &version.ID
		lockedDocument.LatestDraftUpdatedAt = &now
		lockedDocument.UpdatedAt = now

		if err := txRepo.Document.Update(ctx, lockedDocument); err != nil {
			return err
		}

		createdVersionID = version.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	freshDocument, err := s.repos.Document.GetByID(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}

	createdVersion, err := s.repos.DocumentVersion.GetByID(ctx, createdVersionID)
	if err != nil {
		return nil, err
	}

	result := &DocumentVersionResult{
		Document: freshDocument,
		Version:  createdVersion,
	}

	s.publishDocumentUpserted(ctx, result.Document, result.Version)

	return result, nil
}

func (s *contentService) GetDocumentVersionByID(ctx context.Context, requesterID, versionID uuid.UUID) (*model.DocumentVersion, error) {
	version, err := s.repos.DocumentVersion.GetByID(ctx, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentVersionNotFound
		}
		return nil, err
	}

	document, err := s.repos.Document.GetByID(ctx, version.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, document.RepoID, requesterID); err != nil {
		return nil, err
	}

	return version, nil
}

func (s *contentService) ListDocumentVersions(ctx context.Context, requesterID, documentID uuid.UUID, pagination Pagination) ([]*model.DocumentVersion, int64, error) {
	document, err := s.repos.Document.GetByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, domain.ErrDocumentNotFound
		}
		return nil, 0, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, document.RepoID, requesterID); err != nil {
		return nil, 0, err
	}

	return s.repos.DocumentVersion.ListByDocumentID(ctx, documentID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (s *contentService) RestoreDocumentVersion(ctx context.Context, input RestoreDocumentVersionInput) (*DocumentVersionResult, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	document, err := s.repos.Document.GetByID(ctx, input.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, document.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var createdVersionID uuid.UUID

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		lockedDocument, err := txRepo.Document.GetByIDForUpdate(ctx, document.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrDocumentNotFound
			}
			return err
		}

		sourceVersion, err := txRepo.DocumentVersion.GetByDocumentAndVersionID(ctx, lockedDocument.ID, input.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrDocumentVersionNotFound
			}
			return err
		}

		latestNumber, err := txRepo.DocumentVersion.GetLatestVersionNumber(ctx, lockedDocument.ID)
		if err != nil {
			return err
		}

		summary := nullableTrimmedString(input.ChangeSummary)
		if summary == nil {
			defaultSummary := "restored from version " + uintToString(sourceVersion.VersionNumber)
			summary = &defaultSummary
		}

		version := &model.DocumentVersion{
			DocumentID:    lockedDocument.ID,
			AuthorID:      input.RequesterID,
			VersionNumber: latestNumber + 1,
			Content:       sourceVersion.Content,
			ChangeSummary: summary,
			CreatedAt:     now,
		}
		if err := txRepo.DocumentVersion.Create(ctx, version); err != nil {
			return err
		}

		draft := &model.DocumentDraft{
			DocumentID: lockedDocument.ID,
			Content:    sourceVersion.Content,
			UpdatedBy:  input.RequesterID,
			UpdatedAt:  now,
		}
		if err := txRepo.DocumentDraft.Upsert(ctx, draft); err != nil {
			return err
		}

		lockedDocument.CurrentVersionID = &version.ID
		lockedDocument.LatestDraftUpdatedAt = &now
		lockedDocument.UpdatedAt = now

		if err := txRepo.Document.Update(ctx, lockedDocument); err != nil {
			return err
		}

		createdVersionID = version.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	freshDocument, err := s.repos.Document.GetByID(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}

	createdVersion, err := s.repos.DocumentVersion.GetByID(ctx, createdVersionID)
	if err != nil {
		return nil, err
	}

	result := &DocumentVersionResult{
		Document: freshDocument,
		Version:  createdVersion,
	}

	s.publishDocumentUpserted(ctx, result.Document, result.Version)

	return result, nil
}

func (s *contentService) DeleteDocument(ctx context.Context, requesterID, documentID uuid.UUID) error {
	if requesterID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	document, err := s.repos.Document.GetByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrDocumentNotFound
		}
		return err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, document.RepoID, requesterID); err != nil {
		return err
	}

	if err := s.repos.Document.DeleteByID(ctx, documentID); err != nil {
		return err
	}

	if err := s.publisher.PublishDocumentDeleted(ctx, documentID); err != nil {
		s.logger.Warn("failed to publish document deleted event",
			zap.String("document_id", documentID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (s *contentService) CreateFile(ctx context.Context, input CreateFileInput) (*FileState, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}
	if input.RepoID == uuid.Nil {
		return nil, domain.ErrRepositoryNotFound
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, input.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		fileName = fallbackFileNameFromStorageKey(input.StorageKey)
	}
	if fileName == "" {
		fileName = "file"
	}

	now := time.Now().UTC()
	changeSummary := nullableTrimmedString(input.ChangeSummary)
	checksum := nullableTrimmedString(input.ChecksumSHA256)

	var createdFileID uuid.UUID

	err := s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		file := &model.File{
			RepoID:     input.RepoID,
			UploadedBy: input.RequesterID,
			FileName:   fileName,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := txRepo.File.Create(ctx, file); err != nil {
			return err
		}

		version := &model.FileVersion{
			FileID:         file.ID,
			UploadedBy:     input.RequesterID,
			VersionNumber:  1,
			StorageKey:     strings.TrimSpace(input.StorageKey),
			MimeType:       strings.TrimSpace(input.MimeType),
			SizeBytes:      input.SizeBytes,
			ChecksumSHA256: checksum,
			ChangeSummary:  changeSummary,
			CreatedAt:      now,
		}
		if err := txRepo.FileVersion.Create(ctx, version); err != nil {
			return err
		}

		file.CurrentVersionID = &version.ID
		file.UpdatedAt = now

		if err := txRepo.File.Update(ctx, file); err != nil {
			return err
		}

		createdFileID = file.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	state, err := s.GetFileByID(ctx, input.RequesterID, createdFileID)
	if err != nil {
		return nil, err
	}

	s.publishFileUpserted(ctx, state.File, state.CurrentVersion)

	return state, nil
}

func (s *contentService) GetFileByID(ctx context.Context, requesterID, fileID uuid.UUID) (*FileState, error) {
	file, err := s.repos.File.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, file.RepoID, requesterID); err != nil {
		return nil, err
	}

	return &FileState{
		File:           file,
		CurrentVersion: file.CurrentVersion,
	}, nil
}

func (s *contentService) ListRepositoryFiles(ctx context.Context, requesterID, repoID uuid.UUID, pagination Pagination) ([]*model.File, int64, error) {
	if repoID == uuid.Nil {
		return nil, 0, domain.ErrRepositoryNotFound
	}

	if err := s.repoAccess.EnsureCanRead(ctx, repoID, requesterID); err != nil {
		return nil, 0, err
	}

	return s.repos.File.ListByRepoID(ctx, repoID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (s *contentService) AddFileVersion(ctx context.Context, input AddFileVersionInput) (*FileVersionResult, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	file, err := s.repos.File.GetByID(ctx, input.FileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, file.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	changeSummary := nullableTrimmedString(input.ChangeSummary)
	checksum := nullableTrimmedString(input.ChecksumSHA256)

	var createdVersionID uuid.UUID

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		lockedFile, err := txRepo.File.GetByIDForUpdate(ctx, file.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrFileNotFound
			}
			return err
		}

		latestNumber, err := txRepo.FileVersion.GetLatestVersionNumber(ctx, lockedFile.ID)
		if err != nil {
			return err
		}

		version := &model.FileVersion{
			FileID:         lockedFile.ID,
			UploadedBy:     input.RequesterID,
			VersionNumber:  latestNumber + 1,
			StorageKey:     strings.TrimSpace(input.StorageKey),
			MimeType:       strings.TrimSpace(input.MimeType),
			SizeBytes:      input.SizeBytes,
			ChecksumSHA256: checksum,
			ChangeSummary:  changeSummary,
			CreatedAt:      now,
		}
		if err := txRepo.FileVersion.Create(ctx, version); err != nil {
			return err
		}

		lockedFile.CurrentVersionID = &version.ID
		lockedFile.UpdatedAt = now

		if err := txRepo.File.Update(ctx, lockedFile); err != nil {
			return err
		}

		createdVersionID = version.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	freshFile, err := s.repos.File.GetByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	createdVersion, err := s.repos.FileVersion.GetByID(ctx, createdVersionID)
	if err != nil {
		return nil, err
	}

	result := &FileVersionResult{
		File:    freshFile,
		Version: createdVersion,
	}

	s.publishFileUpserted(ctx, result.File, result.Version)

	return result, nil
}

func (s *contentService) GetFileVersionByID(ctx context.Context, requesterID, versionID uuid.UUID) (*model.FileVersion, error) {
	version, err := s.repos.FileVersion.GetByID(ctx, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileVersionNotFound
		}
		return nil, err
	}

	file, err := s.repos.File.GetByID(ctx, version.FileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, file.RepoID, requesterID); err != nil {
		return nil, err
	}

	return version, nil
}

func (s *contentService) ListFileVersions(ctx context.Context, requesterID, fileID uuid.UUID, pagination Pagination) ([]*model.FileVersion, int64, error) {
	file, err := s.repos.File.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, domain.ErrFileNotFound
		}
		return nil, 0, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, file.RepoID, requesterID); err != nil {
		return nil, 0, err
	}

	return s.repos.FileVersion.ListByFileID(ctx, fileID, repository.ListParams{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (s *contentService) RestoreFileVersion(ctx context.Context, input RestoreFileVersionInput) (*FileVersionResult, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	file, err := s.repos.File.GetByID(ctx, input.FileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, file.RepoID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var createdVersionID uuid.UUID

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		lockedFile, err := txRepo.File.GetByIDForUpdate(ctx, file.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrFileNotFound
			}
			return err
		}

		sourceVersion, err := txRepo.FileVersion.GetByFileAndVersionID(ctx, lockedFile.ID, input.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrFileVersionNotFound
			}
			return err
		}

		latestNumber, err := txRepo.FileVersion.GetLatestVersionNumber(ctx, lockedFile.ID)
		if err != nil {
			return err
		}

		summary := nullableTrimmedString(input.ChangeSummary)
		if summary == nil {
			defaultSummary := "restored from version " + uintToString(sourceVersion.VersionNumber)
			summary = &defaultSummary
		}

		version := &model.FileVersion{
			FileID:         lockedFile.ID,
			UploadedBy:     input.RequesterID,
			VersionNumber:  latestNumber + 1,
			StorageKey:     sourceVersion.StorageKey,
			MimeType:       sourceVersion.MimeType,
			SizeBytes:      sourceVersion.SizeBytes,
			ChecksumSHA256: sourceVersion.ChecksumSHA256,
			ChangeSummary:  summary,
			CreatedAt:      now,
		}
		if err := txRepo.FileVersion.Create(ctx, version); err != nil {
			return err
		}

		lockedFile.CurrentVersionID = &version.ID
		lockedFile.UpdatedAt = now

		if err := txRepo.File.Update(ctx, lockedFile); err != nil {
			return err
		}

		createdVersionID = version.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	freshFile, err := s.repos.File.GetByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	createdVersion, err := s.repos.FileVersion.GetByID(ctx, createdVersionID)
	if err != nil {
		return nil, err
	}

	result := &FileVersionResult{
		File:    freshFile,
		Version: createdVersion,
	}

	s.publishFileUpserted(ctx, result.File, result.Version)

	return result, nil
}

func (s *contentService) DeleteFile(ctx context.Context, requesterID, fileID uuid.UUID) error {
	if requesterID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	file, err := s.repos.File.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrFileNotFound
		}
		return err
	}

	if err := s.repoAccess.EnsureCanWrite(ctx, file.RepoID, requesterID); err != nil {
		return err
	}

	if err := s.repos.File.DeleteByID(ctx, fileID); err != nil {
		return err
	}

	if err := s.publisher.PublishFileDeleted(ctx, fileID); err != nil {
		s.logger.Warn("failed to publish file deleted event",
			zap.String("file_id", fileID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (s *contentService) GetFileStorageInfo(ctx context.Context, requesterID, fileID uuid.UUID) (*FileStorageInfo, error) {
	file, err := s.repos.File.GetByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}

	if err := s.repoAccess.EnsureCanRead(ctx, file.RepoID, requesterID); err != nil {
		return nil, err
	}

	if file.CurrentVersion == nil {
		return &FileStorageInfo{}, nil
	}

	storageKey := strings.TrimSpace(file.CurrentVersion.StorageKey)
	if storageKey == "" {
		return &FileStorageInfo{}, nil
	}

	otherCount, err := s.repos.FileVersion.CountByStorageKeyExcludingFile(ctx, storageKey, file.ID)
	if err != nil {
		return nil, err
	}

	return &FileStorageInfo{
		StorageKey:           storageKey,
		OtherReferencesCount: otherCount,
	}, nil
}

func (s *contentService) publishDocumentUpserted(ctx context.Context, document *model.Document, version *model.DocumentVersion) {
	if document == nil {
		return
	}

	metadata, err := s.repoAccess.GetRepositoryMetadata(ctx, document.RepoID)
	if err != nil {
		s.logger.Warn("failed to load repository metadata for document event",
			zap.String("document_id", document.ID.String()),
			zap.String("repo_id", document.RepoID.String()),
			zap.Error(err),
		)
		return
	}

	content := ""
	if version != nil {
		content = version.Content
	}

	if err := s.publisher.PublishDocumentUpserted(ctx, document, content, metadata); err != nil {
		s.logger.Warn("failed to publish document upsert event",
			zap.String("document_id", document.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *contentService) publishFileUpserted(ctx context.Context, file *model.File, version *model.FileVersion) {
	if file == nil {
		return
	}

	metadata, err := s.repoAccess.GetRepositoryMetadata(ctx, file.RepoID)
	if err != nil {
		s.logger.Warn("failed to load repository metadata for file event",
			zap.String("file_id", file.ID.String()),
			zap.String("repo_id", file.RepoID.String()),
			zap.Error(err),
		)
		return
	}

	mimeType := ""
	if version != nil {
		mimeType = version.MimeType
	}

	if err := s.publisher.PublishFileUpserted(ctx, file, mimeType, metadata); err != nil {
		s.logger.Warn("failed to publish file upsert event",
			zap.String("file_id", file.ID.String()),
			zap.Error(err),
		)
	}
}

func normalizeDocumentFormat(format model.DocumentFormat) model.DocumentFormat {
	if format == "" {
		return model.DocumentFormatMarkdown
	}
	return format
}

func validateDocumentFormat(format model.DocumentFormat) error {
	switch format {
	case model.DocumentFormatMarkdown:
		return nil
	default:
		return domain.ErrInvalidDocumentFormat
	}
}

var documentSlugCleanupRegex = regexp.MustCompile(`[^a-z0-9-]+`)
var documentMultiDashRegex = regexp.MustCompile(`-+`)
var cyrillicToLatinReplacer = strings.NewReplacer(
	"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e", "ё", "e",
	"ж", "zh", "з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m",
	"н", "n", "о", "o", "п", "p", "р", "r", "с", "s", "т", "t", "у", "u",
	"ф", "f", "х", "h", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "sch", "ъ", "",
	"ы", "y", "ь", "", "э", "e", "ю", "yu", "я", "ya",
)

func normalizeDocumentSlug(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = cyrillicToLatinReplacer.Replace(v)
	v = strings.ReplaceAll(v, " ", "-")
	v = documentSlugCleanupRegex.ReplaceAllString(v, "-")
	v = documentMultiDashRegex.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")

	if len(v) > 100 {
		v = v[:100]
		v = strings.Trim(v, "-")
	}

	return v
}

func slugifyDocument(title string) string {
	return normalizeDocumentSlug(title)
}

func (s *contentService) findAvailableDocumentSlug(ctx context.Context, repoID uuid.UUID, baseSlug string) (string, error) {
	baseSlug = normalizeDocumentSlug(baseSlug)
	if baseSlug == "" {
		baseSlug = "document"
	}

	for i := 0; i < 10000; i++ {
		candidate := baseSlug
		if i > 0 {
			suffix := fmt.Sprintf("-%d", i+1)
			trimmedBase := baseSlug
			if len(trimmedBase)+len(suffix) > 100 {
				trimmedBase = strings.Trim(trimmedBase[:100-len(suffix)], "-")
				if trimmedBase == "" {
					trimmedBase = "document"
				}
			}
			candidate = trimmedBase + suffix
		}

		_, err := s.repos.Document.GetByRepoAndSlug(ctx, repoID, candidate)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}

	return "", domain.ErrDocumentSlugTaken
}

func fallbackFileNameFromStorageKey(storageKey string) string {
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return ""
	}

	storageKey = strings.ReplaceAll(storageKey, "\\", "/")
	parts := strings.Split(storageKey, "/")
	if len(parts) == 0 {
		return ""
	}

	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return ""
	}

	return name
}

func nullableTrimmedString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func uintToString(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mapDocumentPersistenceError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "uq_documents_repo_slug_active"),
		strings.Contains(msg, "duplicate key"),
		strings.Contains(msg, "repo_id") && strings.Contains(msg, "slug"):
		return domain.ErrDocumentSlugTaken
	default:
		return nil
	}
}
