package drive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	ignore "github.com/sabhiram/go-git-ignore"
	"github.com/soniakeys/graph"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const DefaultIgnoreFile = ".gdriveignore"

type ModTime int

const (
	LocalLastModified ModTime = iota
	RemoteLastModified
	EqualModifiedTime
)

type LargestSize int

const (
	LocalLargestSize LargestSize = iota
	RemoteLargestSize
	EqualSize
)

type ConflictResolution int

const (
	NoResolution ConflictResolution = iota
	KeepLocal
	KeepRemote
	KeepLargest
)

func (g *Drive) prepareSyncFiles(localPath string, root *drive.File, cmp FileComparer) (*syncFiles, error) {
	localCh := make(chan struct {
		files []*LocalFile
		err   error
	})
	remoteCh := make(chan struct {
		files []*RemoteFile
		err   error
	})

	go func() {
		files, err := prepareLocalFiles(localPath)
		localCh <- struct {
			files []*LocalFile
			err   error
		}{files, err}
	}()

	go func() {
		files, err := g.prepareRemoteFiles(root, "")
		remoteCh <- struct {
			files []*RemoteFile
			err   error
		}{files, err}
	}()

	local := <-localCh
	if local.err != nil {
		return nil, local.err
	}

	remote := <-remoteCh
	if remote.err != nil {
		return nil, remote.err
	}

	return &syncFiles{
		root:    &RemoteFile{file: root},
		local:   local.files,
		remote:  remote.files,
		compare: cmp,
	}, nil
}

func (g *Drive) isSyncFile(id string) (bool, error) {
	f, err := g.service.Files.Get(id).Fields("appProperties").Do()
	if err != nil {
		return false, fmt.Errorf("failed to get file: %s", err)
	}

	_, ok := f.AppProperties["sync"]
	return ok, nil
}

func prepareLocalFiles(root string) ([]*LocalFile, error) {
	var files []*LocalFile

	// Get absolute root path
	absRootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Prepare ignorer
	shouldIgnore, err := prepareIgnorer(filepath.Join(absRootPath, DefaultIgnoreFile))
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absRootPath, func(absPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if absPath == absRootPath {
			return nil
		}

		// Skip files that are not a directory or regular file
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil
		}

		// Get relative path from root
		relPath, err := filepath.Rel(absRootPath, absPath)
		if err != nil {
			return err
		}

		// Skip file if it is ignored by ignore file
		if shouldIgnore(relPath) {
			return nil
		}

		files = append(files, &LocalFile{
			absPath: absPath,
			relPath: relPath,
			info:    info,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to prepare local files: %s", err)
	}

	return files, err
}

func (g *Drive) prepareRemoteFiles(rootDir *drive.File, sortOrder string) ([]*RemoteFile, error) {
	// Find all files which has rootDir as root
	listArgs := listAllFilesArgs{
		query:     fmt.Sprintf("appProperties has {key='syncRootId' and value='%s'}", rootDir.Id),
		fields:    []googleapi.Field{"nextPageToken", "files(id,name,parents,md5Checksum,mimeType,size,modifiedTime)"},
		sortOrder: sortOrder,
	}
	files, err := g.listAllFiles(listArgs)
	if err != nil {
		return nil, fmt.Errorf("failed listing files: %s", err)
	}

	if err := checkFiles(files); err != nil {
		return nil, err
	}

	relPaths, err := prepareRemoteRelPaths(rootDir, files)
	if err != nil {
		return nil, err
	}

	var remoteFiles []*RemoteFile
	for _, f := range files {
		relPath, ok := relPaths[f.Id]
		if !ok {
			return nil, fmt.Errorf("file %s does not have a valid parent", f.Id)
		}
		remoteFiles = append(remoteFiles, &RemoteFile{
			relPath: relPath,
			file:    f,
		})
	}

	return remoteFiles, nil
}

func prepareRemoteRelPaths(root *drive.File, files []*drive.File) (map[string]string, error) {
	// The tree only holds integer values so we use
	// maps to lookup file by index and index by file id
	indexLookup := map[string]graph.NI{}
	fileLookup := map[graph.NI]*drive.File{}

	// All files includes root dir
	allFiles := append([]*drive.File{root}, files...)

	// Prepare lookup maps
	for i, f := range allFiles {
		indexLookup[f.Id] = graph.NI(i)
		fileLookup[graph.NI(i)] = f
	}

	// This will hold 'parent index' -> 'file index' relationships
	pathEnds := make([]graph.PathEnd, len(allFiles))

	// Prepare parent -> file relationships
	for i, f := range allFiles {
		if f == root {
			pathEnds[i] = graph.PathEnd{From: -1}
			continue
		}

		// Lookup index of parent
		parentIdx, found := indexLookup[f.Parents[0]]
		if !found {
			return nil, fmt.Errorf("could not find parent of %s (%s)", f.Id, f.Name)
		}
		pathEnds[i] = graph.PathEnd{From: parentIdx}
	}

	// Create parent pointer tree and calculate path lengths
	tree := &graph.FromList{Paths: pathEnds}
	tree.RecalcLeaves()
	tree.RecalcLen()

	// This will hold a map of file id => relative path
	paths := map[string]string{}

	// Find relative path from root for all files
	for _, f := range allFiles {
		if f == root {
			continue
		}

		// Find nodes between root and file
		nodes := tree.PathTo(indexLookup[f.Id], nil)

		// This will hold the name of all paths between root and
		// file (exluding root and including file itself)
		var pathNames []string

		// Lookup file for each node and grab name
		for _, n := range nodes {
			file := fileLookup[n]
			if file == root {
				continue
			}
			pathNames = append(pathNames, file.Name)
		}

		// Join path names to form relative path and add to map
		paths[f.Id] = filepath.Join(pathNames...)
	}

	return paths, nil
}

func checkFiles(files []*drive.File) error {
	uniq := map[string]string{}

	for _, f := range files {
		// Ensure all files have exactly one parent
		if len(f.Parents) != 1 {
			return fmt.Errorf("file %s does not have exacly one parent", f.Id)
		}

		// Ensure that there are no duplicate files
		uniqKey := f.Name + f.Parents[0]
		if dupeId, isDupe := uniq[uniqKey]; isDupe {
			return fmt.Errorf("found name collision between %s and %s", f.Id, dupeId)
		}
		uniq[uniqKey] = f.Id
	}

	return nil
}

type LocalFile struct {
	absPath string
	relPath string
	info    os.FileInfo
}

type RemoteFile struct {
	relPath string
	file    *drive.File
}

type changedFile struct {
	local  *LocalFile
	remote *RemoteFile
}

type syncFiles struct {
	root    *RemoteFile
	local   []*LocalFile
	remote  []*RemoteFile
	compare FileComparer
}

type FileComparer interface {
	Changed(*LocalFile, *RemoteFile) bool
}

func (loc LocalFile) AbsPath() string {
	return loc.absPath
}

func (loc LocalFile) Size() int64 {
	return loc.info.Size()
}

func (loc LocalFile) Modified() time.Time {
	return loc.info.ModTime()
}

func (r RemoteFile) Md5() string {
	return r.file.Md5Checksum
}

func (r RemoteFile) Size() int64 {
	return r.file.Size
}

func (r RemoteFile) Modified() time.Time {
	t, _ := time.Parse(time.RFC3339, r.file.ModifiedTime)
	return t
}

func (c *changedFile) compareModTime() ModTime {
	localTime := c.local.Modified()
	remoteTime := c.remote.Modified()

	if localTime.After(remoteTime) {
		return LocalLastModified
	}

	if remoteTime.After(localTime) {
		return RemoteLastModified
	}

	return EqualModifiedTime
}

func (c *changedFile) compareSize() LargestSize {
	localSize := c.local.Size()
	remoteSize := c.remote.Size()

	if localSize > remoteSize {
		return LocalLargestSize
	}

	if remoteSize > localSize {
		return RemoteLargestSize
	}

	return EqualSize
}

func (s *syncFiles) filterMissingRemoteDirs() []*LocalFile {
	var files []*LocalFile

	for _, lf := range s.local {
		if lf.info.IsDir() && !s.existsRemote(lf) {
			files = append(files, lf)
		}
	}

	return files
}

func (s *syncFiles) filterMissingLocalDirs() []*RemoteFile {
	var files []*RemoteFile

	for _, rf := range s.remote {
		if isDir(rf.file) && !s.existsLocal(rf) {
			files = append(files, rf)
		}
	}

	return files
}

func (s *syncFiles) filterMissingRemoteFiles() []*LocalFile {
	var files []*LocalFile

	for _, lf := range s.local {
		if !lf.info.IsDir() && !s.existsRemote(lf) {
			files = append(files, lf)
		}
	}

	return files
}

func (s *syncFiles) filterMissingLocalFiles() []*RemoteFile {
	var files []*RemoteFile

	for _, rf := range s.remote {
		if !isDir(rf.file) && !s.existsLocal(rf) {
			files = append(files, rf)
		}
	}

	return files
}

func (s *syncFiles) filterChangedLocalFiles() []*changedFile {
	var files []*changedFile

	for _, lf := range s.local {
		// Skip directories
		if lf.info.IsDir() {
			continue
		}

		// Skip files that don't exist on drive
		rf, found := s.findRemoteByPath(lf.relPath)
		if !found {
			continue
		}

		// Check if file has changed
		if s.compare.Changed(lf, rf) {
			files = append(files, &changedFile{
				local:  lf,
				remote: rf,
			})
		}
	}

	return files
}

func (s *syncFiles) filterChangedRemoteFiles() []*changedFile {
	var files []*changedFile

	for _, rf := range s.remote {
		// Skip directories
		if isDir(rf.file) {
			continue
		}

		// Skip local files that don't exist
		lf, found := s.findLocalByPath(rf.relPath)
		if !found {
			continue
		}

		// Check if file has changed
		if s.compare.Changed(lf, rf) {
			files = append(files, &changedFile{
				local:  lf,
				remote: rf,
			})
		}
	}

	return files
}

func (s *syncFiles) filterExtraneousRemoteFiles() []*RemoteFile {
	var files []*RemoteFile

	for _, rf := range s.remote {
		if !s.existsLocal(rf) {
			files = append(files, rf)
		}
	}

	return files
}

func (s *syncFiles) filterExtraneousLocalFiles() []*LocalFile {
	var files []*LocalFile

	for _, lf := range s.local {
		if !s.existsRemote(lf) {
			files = append(files, lf)
		}
	}

	return files
}

func (s *syncFiles) existsRemote(lf *LocalFile) bool {
	_, found := s.findRemoteByPath(lf.relPath)
	return found
}

func (s *syncFiles) existsLocal(rf *RemoteFile) bool {
	_, found := s.findLocalByPath(rf.relPath)
	return found
}

func (s *syncFiles) findRemoteByPath(relPath string) (*RemoteFile, bool) {
	if relPath == "." {
		return s.root, true
	}

	for _, rf := range s.remote {
		if relPath == rf.relPath {
			return rf, true
		}
	}

	return nil, false
}

func (s *syncFiles) findLocalByPath(relPath string) (*LocalFile, bool) {
	for _, lf := range s.local {
		if relPath == lf.relPath {
			return lf, true
		}
	}

	return nil, false
}

func findLocalConflicts(files []*changedFile) []*changedFile {
	var conflicts []*changedFile

	for _, cf := range files {
		if cf.compareModTime() == LocalLastModified {
			conflicts = append(conflicts, cf)
		}
	}

	return conflicts
}

func findRemoteConflicts(files []*changedFile) []*changedFile {
	var conflicts []*changedFile

	for _, cf := range files {
		if cf.compareModTime() == RemoteLastModified {
			conflicts = append(conflicts, cf)
		}
	}

	return conflicts
}

type byLocalPathLength []*LocalFile

func (b byLocalPathLength) Len() int {
	return len(b)
}

func (b byLocalPathLength) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byLocalPathLength) Less(i, j int) bool {
	return pathLength(b[i].relPath) < pathLength(b[j].relPath)
}

type byRemotePathLength []*RemoteFile

func (b byRemotePathLength) Len() int {
	return len(b)
}

func (b byRemotePathLength) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byRemotePathLength) Less(i, j int) bool {
	return pathLength(b[i].relPath) < pathLength(b[j].relPath)
}

type byRemotePath []*RemoteFile

func (b byRemotePath) Len() int {
	return len(b)
}

func (b byRemotePath) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byRemotePath) Less(i, j int) bool {
	return strings.ToLower(b[i].relPath) < strings.ToLower(b[j].relPath)
}

type ignoreFunc func(string) bool

func prepareIgnorer(path string) (ignoreFunc, error) {
	acceptAll := func(string) bool {
		return false
	}

	if !fileExists(path) {
		return acceptAll, nil
	}

	ignorer, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return acceptAll, fmt.Errorf("failed to prepare ignorer: %s", err)
	}

	return ignorer.MatchesPath, nil
}

func formatConflicts(conflicts []*changedFile, out io.Writer) {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 0, 3, ' ', 0)

	_, _ = fmt.Fprintln(w, "Path\tSize Local\tSize Remote\tModified Local\tModified Remote")

	for _, cf := range conflicts {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			truncateString(cf.local.relPath, 60),
			formatSize(cf.local.Size(), false),
			formatSize(cf.remote.Size(), false),
			cf.local.Modified().Local().Format("Jan _2 2006 15:04:05.000"),
			cf.remote.Modified().Local().Format("Jan _2 2006 15:04:05.000"),
		)
	}

	_ = w.Flush()
}
