package grpcserver

import (
	"content-service/internal/domain"
	"content-service/internal/service"
	"context"

	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ContentHandler struct {
	contentv1.UnimplementedContentServiceServer

	service service.ContentService
	logger  *zap.Logger
}

func NewContentHandler(service service.ContentService, logger *zap.Logger) *ContentHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ContentHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ContentHandler) CreateDocument(ctx context.Context, req *contentv1.CreateDocumentRequest) (*contentv1.CreateDocumentResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "create document: missing claims", domain.ErrUnauthorized)
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	state, err := h.service.CreateDocument(ctx, service.CreateDocumentInput{
		RequesterID:    claims.UserID,
		RepoID:         repoID,
		Title:          req.GetTitle(),
		Slug:           req.GetSlug(),
		Format:         toModelDocumentFormat(req.GetFormat()),
		InitialContent: req.GetInitialContent(),
		ChangeSummary:  req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "create document failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", claims.UserID.String()),
			zap.String("title", req.GetTitle()),
		)
	}

	h.logger.Info("document created",
		zap.String("document_id", state.Document.ID.String()),
		zap.String("repo_id", state.Document.RepoID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &contentv1.CreateDocumentResponse{
		Document:       toProtoDocument(state.Document),
		Draft:          toProtoDocumentDraft(state.Draft),
		CurrentVersion: toProtoDocumentVersion(state.CurrentVersion),
	}, nil
}

func (h *ContentHandler) GetDocumentById(ctx context.Context, req *contentv1.GetDocumentByIdRequest) (*contentv1.GetDocumentResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	state, err := h.service.GetDocumentByID(ctx, requesterID, documentID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get document by id failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.GetDocumentResponse{
		Document:       toProtoDocument(state.Document),
		Draft:          toProtoDocumentDraft(state.Draft),
		CurrentVersion: toProtoDocumentVersion(state.CurrentVersion),
	}, nil
}

func (h *ContentHandler) ListRepositoryDocuments(ctx context.Context, req *contentv1.ListRepositoryDocumentsRequest) (*contentv1.ListDocumentsResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	documents, total, err := h.service.ListRepositoryDocuments(ctx, requesterID, repoID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list repository documents failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.ListDocumentsResponse{
		Documents: toProtoDocuments(documents),
		Total:     uint64(total),
	}, nil
}

func (h *ContentHandler) SaveDocumentDraft(ctx context.Context, req *contentv1.SaveDocumentDraftRequest) (*contentv1.SaveDocumentDraftResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "save document draft: missing claims", domain.ErrUnauthorized)
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	state, err := h.service.SaveDocumentDraft(ctx, service.SaveDocumentDraftInput{
		RequesterID: claims.UserID,
		DocumentID:  documentID,
		Content:     req.GetContent(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "save document draft failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &contentv1.SaveDocumentDraftResponse{
		Document: toProtoDocument(state.Document),
		Draft:    toProtoDocumentDraft(state.Draft),
	}, nil
}

func (h *ContentHandler) CreateDocumentVersion(ctx context.Context, req *contentv1.CreateDocumentVersionRequest) (*contentv1.CreateDocumentVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "create document version: missing claims", domain.ErrUnauthorized)
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	result, err := h.service.CreateDocumentVersion(ctx, service.CreateDocumentVersionInput{
		RequesterID:   claims.UserID,
		DocumentID:    documentID,
		Content:       req.GetContent(),
		ChangeSummary: req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "create document version failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &contentv1.CreateDocumentVersionResponse{
		Document: toProtoDocument(result.Document),
		Version:  toProtoDocumentVersion(result.Version),
	}, nil
}

func (h *ContentHandler) GetDocumentVersionById(ctx context.Context, req *contentv1.GetDocumentVersionByIdRequest) (*contentv1.GetDocumentVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	versionID, err := parseProtoUUID(req.GetVersionId(), "version_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	version, err := h.service.GetDocumentVersionByID(ctx, requesterID, versionID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get document version by id failed", err,
			zap.String("version_id", versionID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.GetDocumentVersionResponse{
		Version: toProtoDocumentVersion(version),
	}, nil
}

func (h *ContentHandler) ListDocumentVersions(ctx context.Context, req *contentv1.ListDocumentVersionsRequest) (*contentv1.ListDocumentVersionsResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	versions, total, err := h.service.ListDocumentVersions(ctx, requesterID, documentID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list document versions failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.ListDocumentVersionsResponse{
		Versions: toProtoDocumentVersions(versions),
		Total:    uint64(total),
	}, nil
}

func (h *ContentHandler) RestoreDocumentVersion(ctx context.Context, req *contentv1.RestoreDocumentVersionRequest) (*contentv1.CreateDocumentVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "restore document version: missing claims", domain.ErrUnauthorized)
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	versionID, err := parseProtoUUID(req.GetVersionId(), "version_id")
	if err != nil {
		return nil, err
	}

	result, err := h.service.RestoreDocumentVersion(ctx, service.RestoreDocumentVersionInput{
		RequesterID:   claims.UserID,
		DocumentID:    documentID,
		VersionID:     versionID,
		ChangeSummary: req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "restore document version failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("version_id", versionID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &contentv1.CreateDocumentVersionResponse{
		Document: toProtoDocument(result.Document),
		Version:  toProtoDocumentVersion(result.Version),
	}, nil
}

func (h *ContentHandler) DeleteDocument(ctx context.Context, req *contentv1.DeleteDocumentRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "delete document: missing claims", domain.ErrUnauthorized)
	}

	documentID, err := parseProtoUUID(req.GetDocumentId(), "document_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteDocument(ctx, claims.UserID, documentID); err != nil {
		return nil, LogAndMapError(h.logger, "delete document failed", err,
			zap.String("document_id", documentID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("document deleted",
		zap.String("document_id", documentID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &emptypb.Empty{}, nil
}

func (h *ContentHandler) CreateFile(ctx context.Context, req *contentv1.CreateFileRequest) (*contentv1.CreateFileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "create file: missing claims", domain.ErrUnauthorized)
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	state, err := h.service.CreateFile(ctx, service.CreateFileInput{
		RequesterID:    claims.UserID,
		RepoID:         repoID,
		FileName:       req.GetFileName(),
		StorageKey:     req.GetStorageKey(),
		MimeType:       req.GetMimeType(),
		SizeBytes:      req.GetSizeBytes(),
		ChecksumSHA256: req.GetChecksumSha256(),
		ChangeSummary:  req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "create file failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", claims.UserID.String()),
			zap.String("file_name", req.GetFileName()),
		)
	}

	h.logger.Info("file created",
		zap.String("file_id", state.File.ID.String()),
		zap.String("repo_id", state.File.RepoID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &contentv1.CreateFileResponse{
		File:           toProtoFile(state.File),
		CurrentVersion: toProtoFileVersion(state.CurrentVersion),
	}, nil
}

func (h *ContentHandler) GetFileById(ctx context.Context, req *contentv1.GetFileByIdRequest) (*contentv1.GetFileResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	fileID, err := parseProtoUUID(req.GetFileId(), "file_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	state, err := h.service.GetFileByID(ctx, requesterID, fileID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get file by id failed", err,
			zap.String("file_id", fileID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.GetFileResponse{
		File:           toProtoFile(state.File),
		CurrentVersion: toProtoFileVersion(state.CurrentVersion),
	}, nil
}

func (h *ContentHandler) ListRepositoryFiles(ctx context.Context, req *contentv1.ListRepositoryFilesRequest) (*contentv1.ListFilesResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	repoID, err := parseProtoUUID(req.GetRepoId(), "repo_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	files, total, err := h.service.ListRepositoryFiles(ctx, requesterID, repoID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list repository files failed", err,
			zap.String("repo_id", repoID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.ListFilesResponse{
		Files: toProtoFiles(files),
		Total: uint64(total),
	}, nil
}

func (h *ContentHandler) AddFileVersion(ctx context.Context, req *contentv1.AddFileVersionRequest) (*contentv1.AddFileVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "add file version: missing claims", domain.ErrUnauthorized)
	}

	fileID, err := parseProtoUUID(req.GetFileId(), "file_id")
	if err != nil {
		return nil, err
	}

	result, err := h.service.AddFileVersion(ctx, service.AddFileVersionInput{
		RequesterID:    claims.UserID,
		FileID:         fileID,
		StorageKey:     req.GetStorageKey(),
		MimeType:       req.GetMimeType(),
		SizeBytes:      req.GetSizeBytes(),
		ChecksumSHA256: req.GetChecksumSha256(),
		ChangeSummary:  req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "add file version failed", err,
			zap.String("file_id", fileID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &contentv1.AddFileVersionResponse{
		File:    toProtoFile(result.File),
		Version: toProtoFileVersion(result.Version),
	}, nil
}

func (h *ContentHandler) GetFileVersionById(ctx context.Context, req *contentv1.GetFileVersionByIdRequest) (*contentv1.GetFileVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	versionID, err := parseProtoUUID(req.GetVersionId(), "version_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	version, err := h.service.GetFileVersionByID(ctx, requesterID, versionID)
	if err != nil {
		return nil, LogAndMapError(h.logger, "get file version by id failed", err,
			zap.String("version_id", versionID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.GetFileVersionResponse{
		Version: toProtoFileVersion(version),
	}, nil
}

func (h *ContentHandler) ListFileVersions(ctx context.Context, req *contentv1.ListFileVersionsRequest) (*contentv1.ListFileVersionsResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	fileID, err := parseProtoUUID(req.GetFileId(), "file_id")
	if err != nil {
		return nil, err
	}

	requesterID := requesterIDFromContext(ctx)

	versions, total, err := h.service.ListFileVersions(ctx, requesterID, fileID, service.Pagination{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "list file versions failed", err,
			zap.String("file_id", fileID.String()),
			zap.String("requester_id", requesterID.String()),
		)
	}

	return &contentv1.ListFileVersionsResponse{
		Versions: toProtoFileVersions(versions),
		Total:    uint64(total),
	}, nil
}

func (h *ContentHandler) RestoreFileVersion(ctx context.Context, req *contentv1.RestoreFileVersionRequest) (*contentv1.AddFileVersionResponse, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "restore file version: missing claims", domain.ErrUnauthorized)
	}

	fileID, err := parseProtoUUID(req.GetFileId(), "file_id")
	if err != nil {
		return nil, err
	}

	versionID, err := parseProtoUUID(req.GetVersionId(), "version_id")
	if err != nil {
		return nil, err
	}

	result, err := h.service.RestoreFileVersion(ctx, service.RestoreFileVersionInput{
		RequesterID:   claims.UserID,
		FileID:        fileID,
		VersionID:     versionID,
		ChangeSummary: req.GetChangeSummary(),
	})
	if err != nil {
		return nil, LogAndMapError(h.logger, "restore file version failed", err,
			zap.String("file_id", fileID.String()),
			zap.String("version_id", versionID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	return &contentv1.AddFileVersionResponse{
		File:    toProtoFile(result.File),
		Version: toProtoFileVersion(result.Version),
	}, nil
}

func (h *ContentHandler) DeleteFile(ctx context.Context, req *contentv1.DeleteFileRequest) (*emptypb.Empty, error) {
	if err := validateProto(req); err != nil {
		return nil, err
	}

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, LogAndMapError(h.logger, "delete file: missing claims", domain.ErrUnauthorized)
	}

	fileID, err := parseProtoUUID(req.GetFileId(), "file_id")
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteFile(ctx, claims.UserID, fileID); err != nil {
		return nil, LogAndMapError(h.logger, "delete file failed", err,
			zap.String("file_id", fileID.String()),
			zap.String("requester_id", claims.UserID.String()),
		)
	}

	h.logger.Info("file deleted",
		zap.String("file_id", fileID.String()),
		zap.String("requester_id", claims.UserID.String()),
	)

	return &emptypb.Empty{}, nil
}

type protoValidator interface {
	ValidateAll() error
}

func validateProto(v protoValidator) error {
	if err := v.ValidateAll(); err != nil {
		return grpcstatus.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func requesterIDFromContext(ctx context.Context) uuid.UUID {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return uuid.Nil
	}
	return claims.UserID
}
