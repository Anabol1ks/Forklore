package grpcserver

import (
	"search-service/internal/model"
	"search-service/internal/service"
	"strings"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
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

func parseOptionalProtoUUID(id *commonv1.UUID, fieldName string) (*uuid.UUID, error) {
	if id == nil {
		return nil, nil
	}

	value := strings.TrimSpace(id.GetValue())
	if value == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "%s must be a valid uuid", fieldName)
	}

	return &parsed, nil
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoSearchEntityType(t model.SearchEntityType) commonv1.SearchEntityType {
	switch t {
	case model.SearchEntityTypeRepository:
		return commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_REPOSITORY
	case model.SearchEntityTypeDocument:
		return commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_DOCUMENT
	case model.SearchEntityTypeFile:
		return commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_FILE
	default:
		return commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_UNSPECIFIED
	}
}

func toModelSearchEntityType(t commonv1.SearchEntityType) model.SearchEntityType {
	switch t {
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_REPOSITORY:
		return model.SearchEntityTypeRepository
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_DOCUMENT:
		return model.SearchEntityTypeDocument
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_FILE:
		return model.SearchEntityTypeFile
	default:
		return ""
	}
}

func toModelSearchEntityTypes(items []commonv1.SearchEntityType) []model.SearchEntityType {
	if len(items) == 0 {
		return nil
	}

	result := make([]model.SearchEntityType, 0, len(items))
	for _, item := range items {
		mapped := toModelSearchEntityType(item)
		if mapped == "" {
			continue
		}
		result = append(result, mapped)
	}

	return result
}

func toProtoSearchHit(hit *service.SearchHit) *searchv1.SearchHit {
	if hit == nil {
		return nil
	}

	var repoID *commonv1.UUID
	if hit.RepoID != nil && *hit.RepoID != uuid.Nil {
		repoID = toProtoUUID(*hit.RepoID)
	}

	var ownerID *commonv1.UUID
	if hit.OwnerID != nil && *hit.OwnerID != uuid.Nil {
		ownerID = toProtoUUID(*hit.OwnerID)
	}

	var tagID *commonv1.UUID
	if hit.TagID != nil && *hit.TagID != uuid.Nil {
		tagID = toProtoUUID(*hit.TagID)
	}

	return &searchv1.SearchHit{
		EntityType:  toProtoSearchEntityType(hit.EntityType),
		EntityId:    toProtoUUID(hit.EntityID),
		RepoId:      repoID,
		OwnerId:     ownerID,
		TagId:       tagID,
		Title:       hit.Title,
		Description: derefString(hit.Description),
		Snippet:     derefString(hit.Snippet),
		Rank:        hit.Rank,
		UpdatedAt:   toProtoTimestamp(hit.UpdatedAt),
	}
}

func toProtoSearchHits(items []*service.SearchHit) []*searchv1.SearchHit {
	result := make([]*searchv1.SearchHit, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoSearchHit(item))
	}
	return result
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
