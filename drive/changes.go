package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
)

type ListChangesArgs struct {
	Out        io.Writer
	PageToken  string
	MaxChanges int64
	Now        bool
	NameWidth  int64
	SkipHeader bool
	JsonOut    bool
}

func (g *Drive) ListChanges(args ListChangesArgs) error {
	if args.Now {
		pageToken, err := g.GetChangesStartPageToken()
		if err != nil {
			return err
		}

		mesg := fmt.Sprintf("Page token: %s\n", pageToken)

		if args.JsonOut {
			if jb, err := json.Marshal(map[string]string{
				"mesg": mesg,
			}); err != nil {
				return err
			} else {
				_, _ = fmt.Fprintln(args.Out, string(jb))
				return nil
			}
		}
		_, _ = fmt.Fprintf(args.Out, mesg)
		return nil
	}

	changeList, err := g.service.Changes.List(args.PageToken).PageSize(args.MaxChanges).RestrictToMyDrive(true).Fields("newStartPageToken", "nextPageToken", "changes(fileId,removed,time,file(id,name,md5Checksum,mimeType,createdTime,modifiedTime))").Do()
	if err != nil {
		return fmt.Errorf("failed listing changes: %s", err)
	}

	if args.JsonOut {
		return PrintChangesJson(PrintChangesArgs{
			Out:        args.Out,
			ChangeList: changeList,
			NameWidth:  int(args.NameWidth),
			SkipHeader: args.SkipHeader,
		})
	}

	PrintChanges(PrintChangesArgs{
		Out:        args.Out,
		ChangeList: changeList,
		NameWidth:  int(args.NameWidth),
		SkipHeader: args.SkipHeader,
	})

	return nil
}

func (g *Drive) GetChangesStartPageToken() (string, error) {
	res, err := g.service.Changes.GetStartPageToken().Do()
	if err != nil {
		return "", fmt.Errorf("failed getting start page token: %s", err)
	}

	return res.StartPageToken, nil
}

type PrintChangesArgs struct {
	Out        io.Writer
	ChangeList *drive.ChangeList
	NameWidth  int
	SkipHeader bool
}

func PrintChanges(args PrintChangesArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		_, _ = fmt.Fprintln(w, "Id\tName\tAction\tTime")
	}

	for _, c := range args.ChangeList.Changes {
		var name string
		var action string

		if c.Removed {
			action = "remove"
		} else {
			name = c.File.Name
			action = "update"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			c.FileId,
			truncateString(name, args.NameWidth),
			action,
			formatDatetime(c.Time),
		)
	}

	if len(args.ChangeList.Changes) > 0 {
		_ = w.Flush()
		pageToken, hasMore := nextChangesPageToken(args.ChangeList)
		_, _ = fmt.Fprintf(args.Out, "\nToken: %s, more: %t\n", pageToken, hasMore)
	} else {
		_, _ = fmt.Fprintln(args.Out, "No changes")
	}
}

func PrintChangesJson(args PrintChangesArgs) error {
	jb, err := json.Marshal(args.ChangeList.Changes)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(args.Out, string(jb))
	return nil
}

func nextChangesPageToken(cl *drive.ChangeList) (string, bool) {
	if cl.NextPageToken != "" {
		return cl.NextPageToken, true
	}

	return cl.NewStartPageToken, false
}
