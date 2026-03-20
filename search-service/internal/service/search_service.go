package service

import (
	"context"
	"search-service/internal/domain"
	"search-service/internal/model"
	"search-service/internal/repository"
	"strings"

	"github.com/google/uuid"
)

type searchService struct {
	repos *repository.Repository
}

func NewSearchService(repos *repository.Repository) SearchService {
	return &searchService{
		repos: repos,
	}
}

func (s *searchService) Search(ctx context.Context, params SearchParams) ([]*SearchHit, int64, error) {
	query := strings.TrimSpace(params.Query)

	hits, total, err := s.repos.Search.Search(ctx, repository.SearchParams{
		Query:       query,
		EntityTypes: params.EntityTypes,
		TagID:       params.TagID,
		OwnerID:     params.OwnerID,
		RepoID:      params.RepoID,
		Limit:       params.Limit,
		Offset:      params.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]*SearchHit, 0, len(hits))
	for _, hit := range hits {
		result = append(result, &SearchHit{
			EntityType:  hit.EntityType,
			EntityID:    hit.EntityID,
			RepoID:      hit.RepoID,
			OwnerID:     hit.OwnerID,
			TagID:       hit.TagID,
			Title:       hit.Title,
			Description: hit.Description,
			Snippet:     hit.Snippet,
			Rank:        hit.Rank,
			UpdatedAt:   hit.UpdatedAt,
		})
	}

	return result, total, nil
}

func (s *searchService) ApplyRepositoryUpsert(ctx context.Context, payload RepositoryUpsertPayload) error {
	if payload.RepoID == uuid.Nil {
		return domain.ErrInvalidSearchQuery
	}

	item := &model.SearchIndexItem{
		EntityType:  model.SearchEntityTypeRepository,
		EntityID:    payload.RepoID,
		RepoID:      uuidPtr(payload.RepoID),
		OwnerID:     uuidPtr(payload.OwnerID),
		TagID:       uuidPtr(payload.TagID),
		Title:       strings.TrimSpace(payload.Title),
		Description: nullableTrimmedString(payload.Description),
		Content:     nil,
		TagName:     nullableTrimmedString(payload.TagName),
		MimeType:    nil,
		IsPublic:    payload.IsPublic,
		UpdatedAt:   payload.UpdatedAt,
	}

	if err := s.repos.Search.UpsertRepository(ctx, item); err != nil {
		return err
	}

	return s.repos.Search.PropagateRepositoryMetadata(ctx, repository.RepositoryMetadataPatch{
		RepoID:   payload.RepoID,
		OwnerID:  payload.OwnerID,
		TagID:    payload.TagID,
		TagName:  payload.TagName,
		IsPublic: payload.IsPublic,
	})
}

func (s *searchService) ApplyRepositoryDeleted(ctx context.Context, payload RepositoryDeletedPayload) error {
	if payload.RepoID == uuid.Nil {
		return nil
	}

	return s.repos.Search.DeleteByRepoID(ctx, payload.RepoID)
}

func (s *searchService) ApplyDocumentUpsert(ctx context.Context, payload DocumentUpsertPayload) error {
	if payload.DocumentID == uuid.Nil {
		return domain.ErrInvalidSearchQuery
	}

	item := &model.SearchIndexItem{
		EntityType:  model.SearchEntityTypeDocument,
		EntityID:    payload.DocumentID,
		RepoID:      uuidPtr(payload.RepoID),
		OwnerID:     uuidPtr(payload.OwnerID),
		TagID:       uuidPtr(payload.TagID),
		Title:       strings.TrimSpace(payload.Title),
		Description: nil,
		Content:     nullableRawString(payload.Content),
		TagName:     nullableTrimmedString(payload.TagName),
		MimeType:    nil,
		IsPublic:    payload.IsPublic,
		UpdatedAt:   payload.UpdatedAt,
	}

	return s.repos.Search.UpsertDocument(ctx, item)
}

func (s *searchService) ApplyDocumentDeleted(ctx context.Context, payload DocumentDeletedPayload) error {
	if payload.DocumentID == uuid.Nil {
		return nil
	}

	return s.repos.Search.DeleteByEntity(ctx, model.SearchEntityTypeDocument, payload.DocumentID)
}

func (s *searchService) ApplyFileUpsert(ctx context.Context, payload FileUpsertPayload) error {
	if payload.FileID == uuid.Nil {
		return domain.ErrInvalidSearchQuery
	}

	item := &model.SearchIndexItem{
		EntityType:  model.SearchEntityTypeFile,
		EntityID:    payload.FileID,
		RepoID:      uuidPtr(payload.RepoID),
		OwnerID:     uuidPtr(payload.OwnerID),
		TagID:       uuidPtr(payload.TagID),
		Title:       strings.TrimSpace(payload.FileName),
		Description: nil,
		Content:     nil,
		TagName:     nullableTrimmedString(payload.TagName),
		MimeType:    nullableTrimmedString(payload.MimeType),
		IsPublic:    payload.IsPublic,
		UpdatedAt:   payload.UpdatedAt,
	}

	return s.repos.Search.UpsertFile(ctx, item)
}

func (s *searchService) ApplyFileDeleted(ctx context.Context, payload FileDeletedPayload) error {
	if payload.FileID == uuid.Nil {
		return nil
	}

	return s.repos.Search.DeleteByEntity(ctx, model.SearchEntityTypeFile, payload.FileID)
}

func nullableTrimmedString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func nullableRawString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func uuidPtr(v uuid.UUID) *uuid.UUID {
	if v == uuid.Nil {
		return nil
	}
	return &v
}
