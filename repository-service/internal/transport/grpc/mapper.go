package grpcserver

import (
	"repository-service/internal/model"
	"strings"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{
		Value: id.String(),
	}
}

func parseProtoUUID(id *commonv1.UUID, fieldName string) (uuid.UUID, error) {
	if id == nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	value := strings.TrimSpace(id.GetValue())
	if value == "" {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, grpcstatus.Errorf(codes.InvalidArgument, "%s must be a valid uuid", fieldName)
	}

	return parsed, nil
}

func parseOptionalProtoUUID(id *commonv1.UUID, fieldName string) (*uuid.UUID, error) {
	if id == nil || strings.TrimSpace(id.GetValue()) == "" {
		return nil, nil
	}

	parsed, err := parseProtoUUID(id, fieldName)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func toProtoDeletedAt(t any) *timestamppb.Timestamp {
	switch v := t.(type) {
	case *time.Time:
		return toProtoTimestampPtr(v)
	case time.Time:
		return toProtoTimestamp(v)
	default:
		return nil
	}
}

func toProtoRepositoryVisibility(v model.RepositoryVisibility) commonv1.RepositoryVisibility {
	switch v {
	case model.RepositoryVisibilityPublic:
		return commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PUBLIC
	case model.RepositoryVisibilityPrivate:
		return commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PRIVATE
	default:
		return commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_UNSPECIFIED
	}
}

func toModelRepositoryVisibility(v commonv1.RepositoryVisibility) model.RepositoryVisibility {
	switch v {
	case commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PUBLIC:
		return model.RepositoryVisibilityPublic
	case commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PRIVATE:
		return model.RepositoryVisibilityPrivate
	default:
		return ""
	}
}

func toProtoRepositoryType(t model.RepositoryType) commonv1.RepositoryType {
	switch t {
	case model.RepositoryTypeArticle:
		return commonv1.RepositoryType_REPOSITORY_TYPE_ARTICLE
	case model.RepositoryTypeNotes:
		return commonv1.RepositoryType_REPOSITORY_TYPE_NOTES
	case model.RepositoryTypeMixed:
		return commonv1.RepositoryType_REPOSITORY_TYPE_MIXED
	default:
		return commonv1.RepositoryType_REPOSITORY_TYPE_UNSPECIFIED
	}
}

func toModelRepositoryType(t commonv1.RepositoryType) model.RepositoryType {
	switch t {
	case commonv1.RepositoryType_REPOSITORY_TYPE_ARTICLE:
		return model.RepositoryTypeArticle
	case commonv1.RepositoryType_REPOSITORY_TYPE_NOTES:
		return model.RepositoryTypeNotes
	case commonv1.RepositoryType_REPOSITORY_TYPE_MIXED:
		return model.RepositoryTypeMixed
	default:
		return ""
	}
}

func toProtoRepositoryTag(tag *model.RepositoryTag) *repositoryv1.RepositoryTag {
	if tag == nil {
		return nil
	}

	return &repositoryv1.RepositoryTag{
		TagId:       toProtoUUID(tag.ID),
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: derefString(tag.Description),
		IsActive:    tag.IsActive,
	}
}

func toProtoRepository(repo *model.Repository) *repositoryv1.Repository {
	if repo == nil {
		return nil
	}

	var parentRepoID *commonv1.UUID
	if repo.ParentRepoID != nil && *repo.ParentRepoID != uuid.Nil {
		parentRepoID = toProtoUUID(*repo.ParentRepoID)
	}

	var deletedAt *timestamppb.Timestamp
	if repo.DeletedAt.Valid {
		deletedAt = timestamppb.New(repo.DeletedAt.Time)
	}

	return &repositoryv1.Repository{
		RepoId:        toProtoUUID(repo.ID),
		OwnerId:       toProtoUUID(repo.OwnerID),
		OwnerUsername: repo.OwnerUsername,
		Name:          repo.Name,
		Slug:          repo.Slug,
		Description:   derefString(repo.Description),
		Visibility:    toProtoRepositoryVisibility(repo.Visibility),
		Type:          toProtoRepositoryType(repo.Type),
		TagId:         toProtoUUID(repo.TagID),
		Tag:           toProtoRepositoryTag(repo.Tag),
		ParentRepoId:  parentRepoID,
		CreatedAt:     toProtoTimestamp(repo.CreatedAt),
		UpdatedAt:     toProtoTimestamp(repo.UpdatedAt),
		DeletedAt:     deletedAt,
	}
}

func toProtoRepositories(repos []*model.Repository) []*repositoryv1.Repository {
	result := make([]*repositoryv1.Repository, 0, len(repos))
	for _, repo := range repos {
		result = append(result, toProtoRepository(repo))
	}
	return result
}

func toProtoTags(tags []*model.RepositoryTag) []*repositoryv1.RepositoryTag {
	result := make([]*repositoryv1.RepositoryTag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, toProtoRepositoryTag(tag))
	}
	return result
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
