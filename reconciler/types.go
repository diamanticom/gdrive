package reconciler

import (
	"github.com/gdrive-org/gdrive/drive"
)

// Spec defines spec used to reconcile local state with remote
type Spec struct {
	// Kind is the spec descriptor
	Kind string
	// ApiVersion is the spec API
	ApiVersion string
	// Policy is the reconciliation policy
	Policy *Policy
	// Cache stores files locally for easier retrieval later
	Cache *Cache
	// Files is a list of files to reconcile
	Files []*File
	// g is client to gdrive
	g *drive.Drive
}

// File identifies file on disk and object id on gdrive
type File struct {
	// Name of the file
	Name string
	// LocalPath of file
	LocalPath string
	// Id of file in gdrive
	Id string
	// RevId is revision ID of file
	RevId string
	// Md5 checksum of file used to reconcile states
	Md5 string
	// remoteName is the name of the file on gdrive
	remoteName string
	// cachedPath is the location of local cache
	cachedPath string
	// g is client to gdrive
	g *drive.Drive
}

// Policy during reconciliation
type Policy struct {
	// IgnoreMd5 ignores md5 sum
	IgnoreMd5 bool
}

// Cache is local cache store
type Cache struct {
	Path string
}
