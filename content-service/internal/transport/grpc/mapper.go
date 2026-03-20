package grpcserver

import (
	"content-service/internal/model"
	"strings"
	"time"

	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{Value: id.String()}
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

func toProtoDocumentFormat(format model.DocumentFormat) commonv1.DocumentFormat {
	switch format {
	case model.DocumentFormatMarkdown:
		return commonv1.DocumentFormat_DOCUMENT_FORMAT_MARKDOWN
	default:
		return commonv1.DocumentFormat_DOCUMENT_FORMAT_UNSPECIFIED
	}
}

func toModelDocumentFormat(format commonv1.DocumentFormat) model.DocumentFormat {
	switch format {
	case commonv1.DocumentFormat_DOCUMENT_FORMAT_MARKDOWN:
		return model.DocumentFormatMarkdown
	default:
		return ""
	}
}

func toProtoDocument(document *model.Document) *contentv1.Document {
	if document == nil {
		return nil
	}

	var currentVersionID *commonv1.UUID
	if document.CurrentVersionID != nil && *document.CurrentVersionID != uuid.Nil {
		currentVersionID = toProtoUUID(*document.CurrentVersionID)
	}

	var deletedAt *timestamppb.Timestamp
	if document.DeletedAt.Valid {
		deletedAt = timestamppb.New(document.DeletedAt.Time)
	}

	return &contentv1.Document{
		DocumentId:           toProtoUUID(document.ID),
		RepoId:               toProtoUUID(document.RepoID),
		AuthorId:             toProtoUUID(document.AuthorID),
		Title:                document.Title,
		Slug:                 document.Slug,
		Format:               toProtoDocumentFormat(document.Format),
		CurrentVersionId:     currentVersionID,
		LatestDraftUpdatedAt: toProtoTimestampPtr(document.LatestDraftUpdatedAt),
		CreatedAt:            toProtoTimestamp(document.CreatedAt),
		UpdatedAt:            toProtoTimestamp(document.UpdatedAt),
		DeletedAt:            deletedAt,
	}
}

func toProtoDocumentDraft(draft *model.DocumentDraft) *contentv1.DocumentDraft {
	if draft == nil {
		return nil
	}

	return &contentv1.DocumentDraft{
		DocumentId: toProtoUUID(draft.DocumentID),
		Content:    draft.Content,
		UpdatedBy:  toProtoUUID(draft.UpdatedBy),
		UpdatedAt:  toProtoTimestamp(draft.UpdatedAt),
	}
}

func toProtoDocumentVersion(version *model.DocumentVersion) *contentv1.DocumentVersion {
	if version == nil {
		return nil
	}

	return &contentv1.DocumentVersion{
		VersionId:     toProtoUUID(version.ID),
		DocumentId:    toProtoUUID(version.DocumentID),
		AuthorId:      toProtoUUID(version.AuthorID),
		VersionNumber: version.VersionNumber,
		Content:       version.Content,
		ChangeSummary: derefString(version.ChangeSummary),
		CreatedAt:     toProtoTimestamp(version.CreatedAt),
	}
}

func toProtoDocuments(items []*model.Document) []*contentv1.Document {
	result := make([]*contentv1.Document, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoDocument(item))
	}
	return result
}

func toProtoDocumentVersions(items []*model.DocumentVersion) []*contentv1.DocumentVersion {
	result := make([]*contentv1.DocumentVersion, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoDocumentVersion(item))
	}
	return result
}

func toProtoFile(file *model.File) *contentv1.File {
	if file == nil {
		return nil
	}

	var currentVersionID *commonv1.UUID
	if file.CurrentVersionID != nil && *file.CurrentVersionID != uuid.Nil {
		currentVersionID = toProtoUUID(*file.CurrentVersionID)
	}

	var deletedAt *timestamppb.Timestamp
	if file.DeletedAt.Valid {
		deletedAt = timestamppb.New(file.DeletedAt.Time)
	}

	return &contentv1.File{
		FileId:           toProtoUUID(file.ID),
		RepoId:           toProtoUUID(file.RepoID),
		UploadedBy:       toProtoUUID(file.UploadedBy),
		FileName:         file.FileName,
		CurrentVersionId: currentVersionID,
		CreatedAt:        toProtoTimestamp(file.CreatedAt),
		UpdatedAt:        toProtoTimestamp(file.UpdatedAt),
		DeletedAt:        deletedAt,
	}
}

func toProtoFileVersion(version *model.FileVersion) *contentv1.FileVersion {
	if version == nil {
		return nil
	}

	return &contentv1.FileVersion{
		VersionId:      toProtoUUID(version.ID),
		FileId:         toProtoUUID(version.FileID),
		UploadedBy:     toProtoUUID(version.UploadedBy),
		VersionNumber:  version.VersionNumber,
		StorageKey:     version.StorageKey,
		MimeType:       version.MimeType,
		SizeBytes:      version.SizeBytes,
		ChecksumSha256: derefString(version.ChecksumSHA256),
		ChangeSummary:  derefString(version.ChangeSummary),
		CreatedAt:      toProtoTimestamp(version.CreatedAt),
	}
}

func toProtoFiles(items []*model.File) []*contentv1.File {
	result := make([]*contentv1.File, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoFile(item))
	}
	return result
}

func toProtoFileVersions(items []*model.FileVersion) []*contentv1.FileVersion {
	result := make([]*contentv1.FileVersion, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoFileVersion(item))
	}
	return result
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
