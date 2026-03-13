package service

import (
	"context"
	"errors"
	"regexp"
	"repository-service/internal/domain"
	"repository-service/internal/model"
	"repository-service/internal/repository"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repositoryService struct {
	repos *repository.Repository
}

func NewRepositoryService(repos *repository.Repository) RepositoryService {
	return &repositoryService{
		repos: repos,
	}
}

func (s *repositoryService) CreateRepository(ctx context.Context, input CreateRepositoryInput) (*model.Repository, error) {
	if input.OwnerID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	if err := validateVisibility(input.Visibility); err != nil {
		return nil, err
	}

	if err := validateRepositoryType(input.Type); err != nil {
		return nil, err
	}

	tag, err := s.repos.Tag.GetByID(ctx, input.TagID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTagNotFound
		}
		return nil, err
	}
	if !tag.IsActive {
		return nil, domain.ErrTagInactive
	}

	name := strings.TrimSpace(input.Name)
	description := nullableTrimmedString(input.Description)

	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, domain.ErrRepositorySlugTaken
	}

	if _, err := s.repos.Repo.GetByOwnerAndSlug(ctx, input.OwnerID, slug); err == nil {
		return nil, domain.ErrRepositorySlugTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	repo := &model.Repository{
		OwnerID:     input.OwnerID,
		TagID:       tag.ID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Visibility:  input.Visibility,
		Type:        input.Type,
	}

	if err := s.repos.Repo.Create(ctx, repo); err != nil {
		if mapped := mapCreateOrUpdateRepoError(err); mapped != nil {
			return nil, mapped
		}
		return nil, err
	}

	created, err := s.repos.Repo.GetByID(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *repositoryService) GetRepositoryByID(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID) (*model.Repository, error) {
	repo, err := s.repos.Repo.GetByID(ctx, repoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRepositoryNotFound
		}
		return nil, err
	}

	if !canReadRepository(requesterID, repo) {
		return nil, domain.ErrRepositoryAccessDenied
	}

	return repo, nil
}

func (s *repositoryService) GetRepositoryBySlug(ctx context.Context, requesterID uuid.UUID, ownerID uuid.UUID, slug string) (*model.Repository, error) {
	repo, err := s.repos.Repo.GetByOwnerAndSlug(ctx, ownerID, normalizeSlug(slug))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRepositoryNotFound
		}
		return nil, err
	}

	if !canReadRepository(requesterID, repo) {
		return nil, domain.ErrRepositoryAccessDenied
	}

	return repo, nil
}

func (s *repositoryService) UpdateRepository(ctx context.Context, input UpdateRepositoryInput) (*model.Repository, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	repo, err := s.repos.Repo.GetByID(ctx, input.RepoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRepositoryNotFound
		}
		return nil, err
	}

	if repo.OwnerID != input.RequesterID {
		return nil, domain.ErrRepositoryAccessDenied
	}

	if input.TagID != nil && *input.TagID != uuid.Nil && *input.TagID != repo.TagID {
		tag, err := s.repos.Tag.GetByID(ctx, *input.TagID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domain.ErrTagNotFound
			}
			return nil, err
		}
		if !tag.IsActive {
			return nil, domain.ErrTagInactive
		}
		repo.TagID = tag.ID
	}

	if v := strings.TrimSpace(input.Name); v != "" {
		repo.Name = v
	}

	if input.Description != "" {
		repo.Description = nullableTrimmedString(input.Description)
	}

	if input.Visibility != "" {
		if err := validateVisibility(input.Visibility); err != nil {
			return nil, err
		}
		repo.Visibility = input.Visibility
	}

	if input.Type != "" {
		if err := validateRepositoryType(input.Type); err != nil {
			return nil, err
		}
		repo.Type = input.Type
	}

	if strings.TrimSpace(input.Slug) != "" {
		newSlug := normalizeSlug(input.Slug)
		if newSlug != repo.Slug {
			if _, err := s.repos.Repo.GetByOwnerAndSlug(ctx, repo.OwnerID, newSlug); err == nil {
				return nil, domain.ErrRepositorySlugTaken
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			repo.Slug = newSlug
		}
	}

	if err := s.repos.Repo.Update(ctx, repo); err != nil {
		if mapped := mapCreateOrUpdateRepoError(err); mapped != nil {
			return nil, mapped
		}
		return nil, err
	}

	updated, err := s.repos.Repo.GetByID(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *repositoryService) DeleteRepository(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID) error {
	if requesterID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	repo, err := s.repos.Repo.GetByID(ctx, repoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrRepositoryNotFound
		}
		return err
	}

	if repo.OwnerID != requesterID {
		return domain.ErrRepositoryAccessDenied
	}

	return s.repos.Repo.DeleteByID(ctx, repoID)
}

func (s *repositoryService) ForkRepository(ctx context.Context, input ForkRepositoryInput) (*model.Repository, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	sourceRepo, err := s.repos.Repo.GetByID(ctx, input.SourceRepoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRepositoryNotFound
		}
		return nil, err
	}

	if sourceRepo.Visibility != model.RepositoryVisibilityPublic {
		return nil, domain.ErrRepositoryCannotBeForked
	}

	if sourceRepo.OwnerID == input.RequesterID {
		return nil, domain.ErrRepositoryCannotBeForked
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = sourceRepo.Name
	}

	description := nullableTrimmedString(input.Description)
	if description == nil {
		description = sourceRepo.Description
	}

	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, domain.ErrRepositorySlugTaken
	}

	if _, err := s.repos.Repo.GetByOwnerAndSlug(ctx, input.RequesterID, slug); err == nil {
		return nil, domain.ErrRepositorySlugTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	repo := &model.Repository{
		OwnerID:      input.RequesterID,
		TagID:        sourceRepo.TagID,
		Name:         name,
		Slug:         slug,
		Description:  description,
		Visibility:   model.RepositoryVisibilityPrivate,
		Type:         sourceRepo.Type,
		ParentRepoID: &sourceRepo.ID,
	}

	if err := s.repos.Repo.Create(ctx, repo); err != nil {
		if mapped := mapCreateOrUpdateRepoError(err); mapped != nil {
			return nil, mapped
		}
		return nil, err
	}

	created, err := s.repos.Repo.GetByID(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *repositoryService) ListMyRepositories(ctx context.Context, ownerID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error) {
	if ownerID == uuid.Nil {
		return nil, 0, domain.ErrUnauthorized
	}

	return s.repos.Repo.ListByOwner(ctx, ownerID, toRepoListParams(pagination))
}

func (s *repositoryService) ListUserRepositories(ctx context.Context, requesterID uuid.UUID, ownerID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error) {
	if ownerID == uuid.Nil {
		return nil, 0, domain.ErrUnauthorized
	}

	if requesterID == ownerID {
		return s.repos.Repo.ListByOwner(ctx, ownerID, toRepoListParams(pagination))
	}

	return s.repos.Repo.ListPublicByOwner(ctx, ownerID, toRepoListParams(pagination))
}

func (s *repositoryService) ListForks(ctx context.Context, requesterID uuid.UUID, repoID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error) {
	repo, err := s.repos.Repo.GetByID(ctx, repoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, domain.ErrRepositoryNotFound
		}
		return nil, 0, err
	}

	if !canReadRepository(requesterID, repo) {
		return nil, 0, domain.ErrRepositoryAccessDenied
	}

	return s.repos.Repo.ListForks(ctx, repoID, toRepoListParams(pagination))
}

func (s *repositoryService) ListRepositoryTags(ctx context.Context) ([]*model.RepositoryTag, error) {
	return s.repos.Tag.ListActive(ctx)
}

func validateVisibility(v model.RepositoryVisibility) error {
	switch v {
	case model.RepositoryVisibilityPublic, model.RepositoryVisibilityPrivate:
		return nil
	default:
		return domain.ErrInvalidRepositoryVisibility
	}
}

func validateRepositoryType(t model.RepositoryType) error {
	switch t {
	case model.RepositoryTypeArticle, model.RepositoryTypeNotes, model.RepositoryTypeMixed:
		return nil
	default:
		return domain.ErrInvalidRepositoryType
	}
}

func canReadRepository(requesterID uuid.UUID, repo *model.Repository) bool {
	if repo == nil {
		return false
	}

	if repo.Visibility == model.RepositoryVisibilityPublic {
		return true
	}

	return requesterID != uuid.Nil && repo.OwnerID == requesterID
}

func toRepoListParams(p Pagination) repository.ListParams {
	return repository.ListParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	}
}

func nullableTrimmedString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

var slugCleanupRegex = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDashRegex = regexp.MustCompile(`-+`)

func normalizeSlug(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "-")
	v = slugCleanupRegex.ReplaceAllString(v, "-")
	v = multiDashRegex.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	return v
}

func slugify(name string) string {
	return normalizeSlug(name)
}

func mapCreateOrUpdateRepoError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "uq_repositories_owner_slug_active"),
		strings.Contains(msg, "duplicate key"),
		strings.Contains(msg, "owner_id") && strings.Contains(msg, "slug"):
		return domain.ErrRepositorySlugTaken

	case strings.Contains(msg, "tag_id"),
		strings.Contains(msg, "repository_tags"):
		return domain.ErrTagNotFound

	default:
		return nil
	}
}
