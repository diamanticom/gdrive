package utils

const (
	Name    = "gdrive"
	Version = "2.1.0"
)

const (
	ClientId             = "367116221053-7n0vf5akeru7on6o2fjinrecpdoe99eg.apps.googleusercontent.com"
	ClientSecret         = "1qsNodXNaWq1mQuBjUjmvhoO"
	TokenFilename        = "token_v2.json"
	DefaultCacheFileName = "file_cache.json"
)

const (
	DefaultMaxFiles        = 1000
	DefaultMaxChanges      = 100
	DefaultNameWidth       = 40
	DefaultPathWidth       = 60
	DefaultUploadChunkSize = 8 * 1024 * 1024
	DefaultTimeout         = 5 * 60
	DefaultQuery           = "trashed = false and 'me' in owners"
	DefaultShareRole       = "reader"
	DefaultShareType       = "anyone"
)

const (
	MimeTypeOctetStream  = "application/octet-stream"
	MimeTypePdf          = "application/pdf"
	MimeTypeFolder       = "application/vnd.google-apps.folder"
	MimeTypePresentation = "application/vnd.google-apps.presentation"
	MimeTypeBinary       = "application/x-executable"
	MimeTypeText         = "text/plain"
)

const (
	AssetOwnerKey = "GDRIVE_ASSET_OWNER"
)
