package main

import (
	"encoding/json"
	"os"

	"github.com/gdrive-org/gdrive/utils"

	"github.com/gdrive-org/gdrive/drive"
)

const MinCacheFileSize = 5 * 1024 * 1024

type Md5Comparer struct{}

func (self Md5Comparer) Changed(local *drive.LocalFile, remote *drive.RemoteFile) bool {
	return remote.Md5() != utils.Md5sum(local.AbsPath())
}

type CachedFileInfo struct {
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Md5      string `json:"md5"`
}

func NewCachedMd5Comparer(path string) CachedMd5Comparer {
	cache := map[string]*CachedFileInfo{}

	f, err := os.Open(path)
	if err == nil {
		_ = json.NewDecoder(f).Decode(&cache)
	}
	_ = f.Close()
	return CachedMd5Comparer{path, cache}
}

type CachedMd5Comparer struct {
	path  string
	cache map[string]*CachedFileInfo
}

func (c CachedMd5Comparer) Changed(local *drive.LocalFile, remote *drive.RemoteFile) bool {
	return remote.Md5() != c.md5(local)
}

func (c CachedMd5Comparer) md5(local *drive.LocalFile) string {
	// See if file exist in cache
	cached, found := c.cache[local.AbsPath()]

	// If found and modification time and size has not changed, return cached md5
	if found && local.Modified().UnixNano() == cached.Modified && local.Size() == cached.Size {
		return cached.Md5
	}

	// Calculate new md5 sum
	md5 := utils.Md5sum(local.AbsPath())

	// Cache file info if file meets size criteria
	if local.Size() > MinCacheFileSize {
		c.cacheAdd(local, md5)
		c.persist()
	}

	return md5
}

func (c CachedMd5Comparer) cacheAdd(lf *drive.LocalFile, md5 string) {
	c.cache[lf.AbsPath()] = &CachedFileInfo{
		Size:     lf.Size(),
		Modified: lf.Modified().UnixNano(),
		Md5:      md5,
	}
}

func (c CachedMd5Comparer) persist() {
	_ = utils.WriteJson(c.path, c.cache)
}
