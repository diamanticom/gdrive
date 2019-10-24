package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
)

type ListRevisionsArgs struct {
	Out         io.Writer
	Id          string
	NameWidth   int64
	SkipHeader  bool
	SizeInBytes bool
	JsonOut     bool
}

func (g *Drive) ListRevisions(args ListRevisionsArgs) error {
	revList, err := g.service.Revisions.List(args.Id).
		Fields("revisions(id,keepForever,size,modifiedTime,originalFilename,md5Checksum)").Do()
	if err != nil {
		return fmt.Errorf("failed listing revisions: %s", err)
	}

	pargs := PrintRevisionListArgs{
		Out:         args.Out,
		Revisions:   revList.Revisions,
		NameWidth:   int(args.NameWidth),
		SkipHeader:  args.SkipHeader,
		SizeInBytes: args.SizeInBytes,
	}

	if args.JsonOut {
		if jb, err := json.Marshal(pargs.Revisions); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	PrintRevisionList(pargs)

	return nil
}

type PrintRevisionListArgs struct {
	Out         io.Writer
	Revisions   []*drive.Revision
	NameWidth   int
	SkipHeader  bool
	SizeInBytes bool
}

func PrintRevisionList(args PrintRevisionListArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		_, _ = fmt.Fprintln(w, "Id\tName\tSize\tModified\tKeepForever")
	}

	for _, rev := range args.Revisions {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			rev.Id,
			truncateString(rev.OriginalFilename, args.NameWidth),
			formatSize(rev.Size, args.SizeInBytes),
			formatDatetime(rev.ModifiedTime),
			formatBool(rev.KeepForever),
		)
	}

	_ = w.Flush()
}
