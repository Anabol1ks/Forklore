package models

import (
	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
)

// ──── Document Models ────

type CreateDocumentRequest struct {
	Title          string `json:"title" binding:"required,min=1,max=200"`
	Slug           string `json:"slug" binding:"max=100"`
	InitialContent string `json:"initial_content"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type SaveDraftRequest struct {
	Content string `json:"content"`
}

type CreateVersionRequest struct {
	Content       string `json:"content"`
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

type RestoreVersionRequest struct {
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

type DocumentResponse struct {
	Document *contentv1.Document `json:"document"`
}

type DocumentListResponse struct {
	Documents []*contentv1.Document `json:"documents"`
	Total     uint64                `json:"total"`
}

// ──── Document Version Models ────

type VersionResponse struct {
	Version *contentv1.DocumentVersion `json:"version"`
}

type VersionListResponse struct {
	Versions []*contentv1.DocumentVersion `json:"versions"`
	Total    uint64                       `json:"total"`
}

// ──── File Models ────

type CreateFileRequest struct {
	FileName       string `json:"file_name" binding:"required,min=1,max=255"`
	StorageKey     string `json:"storage_key" binding:"required,min=1,max=2048"`
	MimeType       string `json:"mime_type" binding:"required,min=1,max=255"`
	SizeBytes      uint64 `json:"size_bytes" binding:"required"`
	ChecksumSHA256 string `json:"checksum_sha256" binding:"max=64"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type AddFileVersionRequest struct {
	StorageKey     string `json:"storage_key" binding:"required,min=1,max=2048"`
	MimeType       string `json:"mime_type" binding:"required,min=1,max=255"`
	SizeBytes      uint64 `json:"size_bytes" binding:"required"`
	ChecksumSHA256 string `json:"checksum_sha256" binding:"max=64"`
	ChangeSummary  string `json:"change_summary" binding:"max=255"`
}

type RestoreFileVersionRequest struct {
	ChangeSummary string `json:"change_summary" binding:"max=255"`
}

type FileResponse struct {
	File *contentv1.File `json:"file"`
}

type FileListResponse struct {
	Files []*contentv1.File `json:"files"`
	Total uint64            `json:"total"`
}

// ──── File Version Models ────

type FileVersionResponse struct {
	Version *contentv1.FileVersion `json:"version"`
}

type FileVersionListResponse struct {
	Versions []*contentv1.FileVersion `json:"versions"`
	Total    uint64                   `json:"total"`
}
