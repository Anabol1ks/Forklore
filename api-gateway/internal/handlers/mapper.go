package handlers

import (
	"api-gateway/internal/models"
	"time"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
)

func mapAuthResponse(resp *authv1.AuthResponse) models.AuthResponse {
	return models.AuthResponse{
		User:   mapUser(resp.GetUser()),
		Tokens: mapTokenPair(resp.GetTokens()),
	}
}

func mapUser(u *authv1.User) models.UserResponse {
	if u == nil {
		return models.UserResponse{}
	}

	var lastLogin *string
	if u.GetLastLoginAt() != nil {
		s := u.GetLastLoginAt().AsTime().Format(time.RFC3339)
		lastLogin = &s
	}

	var updatedAt string
	if u.GetUpdatedAt() != nil {
		updatedAt = u.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	var createdAt string
	if u.GetCreatedAt() != nil {
		createdAt = u.GetCreatedAt().AsTime().Format(time.RFC3339)
	}

	return models.UserResponse{
		UserID:      u.GetUserId().GetValue(),
		Username:    u.GetUsername(),
		Email:       u.GetEmail(),
		Role:        mapUserRole(u.GetRole()),
		Status:      mapUserStatus(u.GetStatus()),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		LastLoginAt: lastLogin,
	}
}

func mapTokenPair(t *authv1.TokenPair) models.TokenPairResponse {
	if t == nil {
		return models.TokenPairResponse{}
	}

	var accessExp, refreshExp string
	if t.GetAccessExpiresAt() != nil {
		accessExp = t.GetAccessExpiresAt().AsTime().Format(time.RFC3339)
	}
	if t.GetRefreshExpiresAt() != nil {
		refreshExp = t.GetRefreshExpiresAt().AsTime().Format(time.RFC3339)
	}

	return models.TokenPairResponse{
		AccessToken:      t.GetAccessToken(),
		RefreshToken:     t.GetRefreshToken(),
		TokenType:        t.GetTokenType(),
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		SessionID:        t.GetSessionId().GetValue(),
	}
}

func mapUserRole(role commonv1.UserRole) string {
	switch role {
	case commonv1.UserRole_USER_ROLE_ADMIN:
		return "admin"
	default:
		return "user"
	}
}

func mapUserStatus(status commonv1.UserStatus) string {
	switch status {
	case commonv1.UserStatus_USER_STATUS_BLOCKED:
		return "blocked"
	case commonv1.UserStatus_USER_STATUS_DELETED:
		return "deleted"
	default:
		return "active"
	}
}

// Repository mapping functions

func mapRepository(r *repositoryv1.Repository) models.RepositoryResponse {
	if r == nil {
		return models.RepositoryResponse{}
	}

	var createdAt, updatedAt, deletedAt string
	if r.GetCreatedAt() != nil {
		createdAt = r.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if r.GetUpdatedAt() != nil {
		updatedAt = r.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	var deletedAtPtr *string
	if r.GetDeletedAt() != nil {
		deletedAt = r.GetDeletedAt().AsTime().Format(time.RFC3339)
		deletedAtPtr = &deletedAt
	}

	var parentRepoID *string
	if r.GetParentRepoId() != nil && r.GetParentRepoId().GetValue() != "" {
		parentRepoID = &r.ParentRepoId.Value
	}

	resp := models.RepositoryResponse{
		RepoID:        r.GetRepoId().GetValue(),
		OwnerID:       r.GetOwnerId().GetValue(),
		OwnerUsername: r.GetOwnerUsername(),
		Name:          r.GetName(),
		Slug:          r.GetSlug(),
		Description:   toPointerString(r.GetDescription()),
		Visibility:    mapRepositoryVisibility(r.GetVisibility()),
		Type:          mapRepositoryType(r.GetType()),
		Tag:           mapRepositoryTag(r.GetTag()),
		ParentRepoID:  parentRepoID,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAtPtr,
	}

	return resp
}

func mapRepositoryTag(t *repositoryv1.RepositoryTag) models.RepositoryTagResponse {
	if t == nil {
		return models.RepositoryTagResponse{}
	}

	var createdAt, updatedAt string
	if t.GetCreatedAt() != nil {
		createdAt = t.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if t.GetUpdatedAt() != nil {
		updatedAt = t.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return models.RepositoryTagResponse{
		TagID:       t.GetTagId().GetValue(),
		Name:        t.GetName(),
		Slug:        t.GetSlug(),
		Description: t.GetDescription(),
		IsActive:    t.GetIsActive(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func mapRepositoryVisibility(v commonv1.RepositoryVisibility) string {
	switch v {
	case commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PRIVATE:
		return "private"
	default:
		return "public"
	}
}

func mapRepositoryType(t commonv1.RepositoryType) string {
	switch t {
	case commonv1.RepositoryType_REPOSITORY_TYPE_NOTES:
		return "notes"
	case commonv1.RepositoryType_REPOSITORY_TYPE_MIXED:
		return "mixed"
	default:
		return "article"
	}
}

// toPointerString converts empty string to nil pointer.
func toPointerString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Content mapping functions

func mapDocument(d *contentv1.Document) models.DocumentDetailResponse {
	if d == nil {
		return models.DocumentDetailResponse{}
	}

	var createdAt, updatedAt, deletedAt, latestDraftUpdatedAt string
	if d.GetCreatedAt() != nil {
		createdAt = d.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if d.GetUpdatedAt() != nil {
		updatedAt = d.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}
	if d.GetDeletedAt() != nil {
		deletedAt = d.GetDeletedAt().AsTime().Format(time.RFC3339)
	}
	if d.GetLatestDraftUpdatedAt() != nil {
		latestDraftUpdatedAt = d.GetLatestDraftUpdatedAt().AsTime().Format(time.RFC3339)
	}

	var updatedAtPtr, deletedAtPtr, latestDraftUpdatedAtPtr *string
	if updatedAt != "" {
		updatedAtPtr = &updatedAt
	}
	if deletedAt != "" {
		deletedAtPtr = &deletedAt
	}
	if latestDraftUpdatedAt != "" {
		latestDraftUpdatedAtPtr = &latestDraftUpdatedAt
	}

	return models.DocumentDetailResponse{
		DocumentID:           d.GetDocumentId().GetValue(),
		RepoID:               d.GetRepoId().GetValue(),
		AuthorID:             d.GetAuthorId().GetValue(),
		Title:                d.GetTitle(),
		Slug:                 d.GetSlug(),
		Format:               mapDocumentFormat(d.GetFormat()),
		CurrentVersionID:     d.GetCurrentVersionId().GetValue(),
		LatestDraftUpdatedAt: latestDraftUpdatedAtPtr,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAtPtr,
		DeletedAt:            deletedAtPtr,
	}
}

func mapDocumentWithDraft(d *contentv1.Document, draft *contentv1.DocumentDraft, version *contentv1.DocumentVersion) models.DocumentDetailResponse {
	doc := mapDocument(d)
	if draft != nil {
		doc.Draft = mapDocumentDraft(draft)
	}
	if version != nil {
		doc.CurrentVersion = mapDocumentVersion(version)
	}
	return doc
}

func mapDocumentDraft(d *contentv1.DocumentDraft) *models.DocumentDraftResponse {
	if d == nil {
		return nil
	}

	var updatedAt string
	if d.GetUpdatedAt() != nil {
		updatedAt = d.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return &models.DocumentDraftResponse{
		DocumentID: d.GetDocumentId().GetValue(),
		Content:    d.GetContent(),
		UpdatedBy:  d.GetUpdatedBy().GetValue(),
		UpdatedAt:  updatedAt,
	}
}

func mapDocumentVersion(v *contentv1.DocumentVersion) *models.DocumentVersionDetail {
	if v == nil {
		return nil
	}

	var createdAt string
	if v.GetCreatedAt() != nil {
		createdAt = v.GetCreatedAt().AsTime().Format(time.RFC3339)
	}

	return &models.DocumentVersionDetail{
		VersionID:     v.GetVersionId().GetValue(),
		DocumentID:    v.GetDocumentId().GetValue(),
		AuthorID:      v.GetAuthorId().GetValue(),
		VersionNumber: v.GetVersionNumber(),
		Content:       v.GetContent(),
		ChangeSummary: v.GetChangeSummary(),
		CreatedAt:     createdAt,
	}
}

func mapDocumentFormat(f commonv1.DocumentFormat) string {
	switch f {
	case commonv1.DocumentFormat_DOCUMENT_FORMAT_MARKDOWN:
		return "markdown"
	default:
		return "markdown"
	}
}

func mapFile(f *contentv1.File) models.FileDetailResponse {
	if f == nil {
		return models.FileDetailResponse{}
	}

	var createdAt, updatedAt, deletedAt string
	if f.GetCreatedAt() != nil {
		createdAt = f.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if f.GetUpdatedAt() != nil {
		updatedAt = f.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}
	if f.GetDeletedAt() != nil {
		deletedAt = f.GetDeletedAt().AsTime().Format(time.RFC3339)
	}

	var updatedAtPtr, deletedAtPtr *string
	if updatedAt != "" {
		updatedAtPtr = &updatedAt
	}
	if deletedAt != "" {
		deletedAtPtr = &deletedAt
	}

	return models.FileDetailResponse{
		FileID:           f.GetFileId().GetValue(),
		RepoID:           f.GetRepoId().GetValue(),
		UploadedBy:       f.GetUploadedBy().GetValue(),
		FileName:         f.GetFileName(),
		CurrentVersionID: f.GetCurrentVersionId().GetValue(),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAtPtr,
		DeletedAt:        deletedAtPtr,
	}
}

func mapFileVersion(v *contentv1.FileVersion) models.FileVersionDetail {
	if v == nil {
		return models.FileVersionDetail{}
	}

	var createdAt string
	if v.GetCreatedAt() != nil {
		createdAt = v.GetCreatedAt().AsTime().Format(time.RFC3339)
	}

	return models.FileVersionDetail{
		VersionID:      v.GetVersionId().GetValue(),
		FileID:         v.GetFileId().GetValue(),
		UploadedBy:     v.GetUploadedBy().GetValue(),
		VersionNumber:  v.GetVersionNumber(),
		StorageKey:     v.GetStorageKey(),
		MimeType:       v.GetMimeType(),
		SizeBytes:      v.GetSizeBytes(),
		ChecksumSHA256: v.GetChecksumSha256(),
		ChangeSummary:  v.GetChangeSummary(),
		CreatedAt:      createdAt,
	}
}

func mapSearchHit(hit *searchv1.SearchHit) models.SearchHitResponse {
	if hit == nil {
		return models.SearchHitResponse{}
	}

	var repoID *string
	if hit.GetRepoId() != nil && hit.GetRepoId().GetValue() != "" {
		value := hit.GetRepoId().GetValue()
		repoID = &value
	}

	var ownerID *string
	if hit.GetOwnerId() != nil && hit.GetOwnerId().GetValue() != "" {
		value := hit.GetOwnerId().GetValue()
		ownerID = &value
	}

	var tagID *string
	if hit.GetTagId() != nil && hit.GetTagId().GetValue() != "" {
		value := hit.GetTagId().GetValue()
		tagID = &value
	}

	var updatedAt string
	if hit.GetUpdatedAt() != nil {
		updatedAt = hit.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return models.SearchHitResponse{
		EntityType:  mapSearchEntityType(hit.GetEntityType()),
		EntityID:    hit.GetEntityId().GetValue(),
		RepoID:      repoID,
		OwnerID:     ownerID,
		TagID:       tagID,
		Title:       hit.GetTitle(),
		Description: toPointerString(hit.GetDescription()),
		Snippet:     toPointerString(hit.GetSnippet()),
		Rank:        hit.GetRank(),
		UpdatedAt:   updatedAt,
	}
}

func mapSearchEntityType(entityType commonv1.SearchEntityType) string {
	switch entityType {
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_REPOSITORY:
		return "repository"
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_DOCUMENT:
		return "document"
	case commonv1.SearchEntityType_SEARCH_ENTITY_TYPE_FILE:
		return "file"
	default:
		return "unspecified"
	}
}
