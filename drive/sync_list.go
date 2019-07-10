package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListSyncArgs struct {
	Out        io.Writer
	SkipHeader bool
	JsonOut    bool
}

func (g *Drive) ListSync(args ListSyncArgs) error {
	listArgs := listAllFilesArgs{
		query:  "appProperties has {key='syncRoot' and value='true'}",
		fields: []googleapi.Field{"nextPageToken", "files(id,name,mimeType,createdTime)"},
	}
	files, err := g.listAllFiles(listArgs)
	if err != nil {
		return err
	}
	if args.JsonOut {
		return printSyncDirectoriesJson(files, args)
	}

	printSyncDirectories(files, args)
	return nil
}

type ListRecursiveSyncArgs struct {
	Out         io.Writer
	RootId      string
	SkipHeader  bool
	PathWidth   int64
	SizeInBytes bool
	SortOrder   string
	JsonOut     bool
}

func (g *Drive) ListRecursiveSync(args ListRecursiveSyncArgs) error {
	rootDir, err := g.getSyncRoot(args.RootId)
	if err != nil {
		return err
	}

	files, err := g.prepareRemoteFiles(rootDir, args.SortOrder)
	if err != nil {
		return err
	}

	if args.JsonOut {
		return printSyncDirContentJson(files, args)
	}

	printSyncDirContent(files, args)
	return nil
}

func printSyncDirectoriesJson(files []*drive.File, args ListSyncArgs) error {
	jb, err := json.Marshal(files)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(args.Out, string(jb))
	return nil
}

func printSyncDirectories(files []*drive.File, args ListSyncArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		_, _ = fmt.Fprintln(w, "Id\tName\tCreated")
	}

	for _, f := range files {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			f.Id,
			f.Name,
			formatDatetime(f.CreatedTime),
		)
	}

	_ = w.Flush()
}

func printSyncDirContentJson(files []*RemoteFile, args ListRecursiveSyncArgs) error {
	jb, err := json.Marshal(files)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(args.Out, string(jb))
	return nil
}

func printSyncDirContent(files []*RemoteFile, args ListRecursiveSyncArgs) {
	if args.SortOrder == "" {
		// Sort files by path
		sort.Sort(byRemotePath(files))
	}

	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		_, _ = fmt.Fprintln(w, "Id\tPath\tType\tSize\tModified")
	}

	for _, rf := range files {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			rf.file.Id,
			truncateString(rf.relPath, int(args.PathWidth)),
			filetype(rf.file),
			formatSize(rf.file.Size, args.SizeInBytes),
			formatDatetime(rf.file.ModifiedTime),
		)
	}

	_ = w.Flush()
}
