package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"repository-service/internal/domain"
	"repository-service/internal/model"
	"repository-service/internal/repository"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repositoryService struct {
	repos     *repository.Repository
	publisher SearchEventPublisher
	logger    *zap.Logger
}

func NewRepositoryService(repos *repository.Repository, publisher SearchEventPublisher, logger *zap.Logger) RepositoryService {
	if publisher == nil {
		publisher = NewNoopSearchEventPublisher()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &repositoryService{
		repos:     repos,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *repositoryService) CreateRepository(ctx context.Context, input CreateRepositoryInput) (*model.Repository, error) {
	if input.OwnerID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}
	if strings.TrimSpace(input.OwnerUsername) == "" {
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

	userProvidedSlug := strings.TrimSpace(input.Slug) != ""
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		slug = "repo"
	}

	if userProvidedSlug {
		if _, err := s.repos.Repo.GetByOwnerAndSlug(ctx, input.OwnerID, slug); err == nil {
			return nil, domain.ErrRepositorySlugTaken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		resolvedSlug, err := s.findAvailableSlug(ctx, input.OwnerID, slug)
		if err != nil {
			return nil, err
		}
		slug = resolvedSlug
	}

	repo := &model.Repository{
		OwnerID:       input.OwnerID,
		OwnerUsername: normalizeOwnerUsername(input.OwnerUsername),
		TagID:         tag.ID,
		Name:          name,
		Slug:          slug,
		Description:   description,
		Visibility:    input.Visibility,
		Type:          input.Type,
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

	if err := s.publisher.PublishRepositoryUpserted(ctx, created); err != nil {
		s.logger.Warn("failed to publish repository upsert event",
			zap.String("repo_id", created.ID.String()),
			zap.Error(err),
		)
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

func (s *repositoryService) GetRepositoryBySlug(ctx context.Context, requesterID uuid.UUID, requesterUsername string, ownerKey string, slug string) (*model.Repository, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil, domain.ErrRepositoryNotFound
	}

	var (
		repo *model.Repository
		err  error
	)

	if ownerUUID, parseErr := uuid.Parse(ownerKey); parseErr == nil {
		repo, err = s.repos.Repo.GetByOwnerAndSlug(ctx, ownerUUID, normalizeSlug(slug))
	} else {
		repo, err = s.repos.Repo.GetByOwnerUsernameAndSlug(ctx, normalizeOwnerUsername(ownerKey), normalizeSlug(slug))
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRepositoryNotFound
		}
		return nil, err
	}

	if !canReadRepository(requesterID, repo) {
		if normalizeOwnerUsername(requesterUsername) != "" && normalizeOwnerUsername(requesterUsername) == repo.OwnerUsername {
			return repo, nil
		}
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

	if err := s.publisher.PublishRepositoryUpserted(ctx, updated); err != nil {
		s.logger.Warn("failed to publish repository upsert event",
			zap.String("repo_id", updated.ID.String()),
			zap.Error(err),
		)
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

	if err := s.repos.Repo.DeleteByID(ctx, repoID); err != nil {
		return err
	}

	if err := s.publisher.PublishRepositoryDeleted(ctx, repoID); err != nil {
		s.logger.Warn("failed to publish repository deleted event",
			zap.String("repo_id", repoID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (s *repositoryService) ForkRepository(ctx context.Context, input ForkRepositoryInput) (*model.Repository, error) {
	if input.RequesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}
	if strings.TrimSpace(input.RequesterUsername) == "" {
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

	userProvidedSlug := strings.TrimSpace(input.Slug) != ""
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		slug = "repo"
	}

	if userProvidedSlug {
		if _, err := s.repos.Repo.GetByOwnerAndSlug(ctx, input.RequesterID, slug); err == nil {
			return nil, domain.ErrRepositorySlugTaken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		resolvedSlug, err := s.findAvailableSlug(ctx, input.RequesterID, slug)
		if err != nil {
			return nil, err
		}
		slug = resolvedSlug
	}

	repo := &model.Repository{
		OwnerID:       input.RequesterID,
		OwnerUsername: normalizeOwnerUsername(input.RequesterUsername),
		TagID:         sourceRepo.TagID,
		Name:          name,
		Slug:          slug,
		Description:   description,
		Visibility:    model.RepositoryVisibilityPrivate,
		Type:          sourceRepo.Type,
		ParentRepoID:  &sourceRepo.ID,
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

	if err := s.publisher.PublishRepositoryUpserted(ctx, created); err != nil {
		s.logger.Warn("failed to publish repository upsert event",
			zap.String("repo_id", created.ID.String()),
			zap.Error(err),
		)
	}

	return created, nil
}

func (s *repositoryService) ListMyRepositories(ctx context.Context, ownerID uuid.UUID, pagination Pagination) ([]*model.Repository, int64, error) {
	if ownerID == uuid.Nil {
		return nil, 0, domain.ErrUnauthorized
	}

	return s.repos.Repo.ListByOwner(ctx, ownerID, toRepoListParams(pagination))
}

func (s *repositoryService) ListUserRepositories(ctx context.Context, requesterID uuid.UUID, requesterUsername string, ownerKey string, pagination Pagination) ([]*model.Repository, int64, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil, 0, domain.ErrUnauthorized
	}

	if ownerUUID, parseErr := uuid.Parse(ownerKey); parseErr == nil {
		if requesterID == ownerUUID {
			return s.repos.Repo.ListByOwner(ctx, ownerUUID, toRepoListParams(pagination))
		}
		return s.repos.Repo.ListPublicByOwner(ctx, ownerUUID, toRepoListParams(pagination))
	}

	normalizedOwner := normalizeOwnerUsername(ownerKey)
	if normalizedOwner == "" {
		return nil, 0, domain.ErrUnauthorized
	}

	if normalizeOwnerUsername(requesterUsername) == normalizedOwner {
		return s.repos.Repo.ListByOwnerUsername(ctx, normalizedOwner, toRepoListParams(pagination))
	}

	return s.repos.Repo.ListPublicByOwnerUsername(ctx, normalizedOwner, toRepoListParams(pagination))
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
var cyrillicToLatinReplacer = strings.NewReplacer(
	"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e", "ё", "e",
	"ж", "zh", "з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m",
	"н", "n", "о", "o", "п", "p", "р", "r", "с", "s", "т", "t", "у", "u",
	"ф", "f", "х", "h", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "sch", "ъ", "",
	"ы", "y", "ь", "", "э", "e", "ю", "yu", "я", "ya",
)

func normalizeSlug(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = cyrillicToLatinReplacer.Replace(v)
	v = strings.ReplaceAll(v, " ", "-")
	v = slugCleanupRegex.ReplaceAllString(v, "-")
	v = multiDashRegex.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	return v
}

func slugify(name string) string {
	return normalizeSlug(name)
}

func normalizeOwnerUsername(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	return v
}

func (s *repositoryService) findAvailableSlug(ctx context.Context, ownerID uuid.UUID, baseSlug string) (string, error) {
	baseSlug = normalizeSlug(baseSlug)
	if baseSlug == "" {
		baseSlug = "repo"
	}

	for i := 0; i < 10000; i++ {
		candidate := baseSlug
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", baseSlug, i+1)
		}

		_, err := s.repos.Repo.GetByOwnerAndSlug(ctx, ownerID, candidate)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}

	return "", domain.ErrRepositorySlugTaken
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
