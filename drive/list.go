package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"golang.org/x/net/context"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListFilesArgs struct {
	Out         io.Writer
	MaxFiles    int64
	NameWidth   int64
	Query       string
	SortOrder   string
	SkipHeader  bool
	SizeInBytes bool
	AbsPath     bool
	JsonOut     bool
}

func (g *Drive) List(args ListFilesArgs) (err error) {
	listArgs := listAllFilesArgs{
		query:     args.Query,
		fields:    []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,createdTime,parents)"},
		sortOrder: args.SortOrder,
		maxFiles:  args.MaxFiles,
	}
	files, err := g.listAllFiles(listArgs)
	if err != nil {
		return fmt.Errorf("failed to list files: %s", err)
	}

	pathfinder := g.newPathfinder()

	if args.AbsPath {
		// Replace name with absolute path
		for _, f := range files {
			f.Name, err = pathfinder.absPath(f)
			if err != nil {
				return err
			}
		}
	}

	p := PrintFileListArgs{
		Out:         args.Out,
		Files:       files,
		NameWidth:   int(args.NameWidth),
		SkipHeader:  args.SkipHeader,
		SizeInBytes: args.SizeInBytes,
	}

	if args.JsonOut {
		if jb, err := json.Marshal(p); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	PrintFileList(p)

	return
}

type listAllFilesArgs struct {
	query     string
	fields    []googleapi.Field
	sortOrder string
	maxFiles  int64
}

func (g *Drive) listAllFiles(args listAllFilesArgs) ([]*drive.File, error) {
	var files []*drive.File

	var pageSize int64
	if args.maxFiles > 0 && args.maxFiles < 1000 {
		pageSize = args.maxFiles
	} else {
		pageSize = 1000
	}

	controlledStop := fmt.Errorf("controlled stop")

	err := g.service.Files.List().Q(args.query).Fields(args.fields...).OrderBy(args.sortOrder).PageSize(pageSize).Pages(context.TODO(), func(fl *drive.FileList) error {
		files = append(files, fl.Files...)

		// Stop when we have all the files we need
		if args.maxFiles > 0 && len(files) >= int(args.maxFiles) {
			return controlledStop
		}

		return nil
	})

	if err != nil && err != controlledStop {
		return nil, err
	}

	if args.maxFiles > 0 {
		n := min(len(files), int(args.maxFiles))
		return files[:n], nil
	}

	return files, nil
}

type PrintFileListArgs struct {
	Out         io.Writer
	Files       []*drive.File
	NameWidth   int
	SkipHeader  bool
	SizeInBytes bool
}

func PrintFileList(args PrintFileListArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		_, _ = fmt.Fprintln(w, "Id\tName\tType\tSize\tCreated")
	}

	for _, f := range args.Files {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			f.Id,
			truncateString(f.Name, args.NameWidth),
			filetype(f),
			formatSize(f.Size, args.SizeInBytes),
			formatDatetime(f.CreatedTime),
		)
	}

	_ = w.Flush()
}

func filetype(f *drive.File) string {
	if isDir(f) {
		return "dir"
	} else if isBinary(f) {
		return "bin"
	}
	return "doc"
}
