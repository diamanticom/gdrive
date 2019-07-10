package drive

import (
	"encoding/json"
	"fmt"
	"io"
)

type DeleteRevisionArgs struct {
	Out        io.Writer
	FileId     string
	RevisionId string
	JsonOut    bool
}

func (g *Drive) DeleteRevision(args DeleteRevisionArgs) error {
	rev, err := g.service.Revisions.Get(args.FileId, args.RevisionId).Fields("originalFilename").Do()
	if err != nil {
		return fmt.Errorf("failed to get revision: %s", err)
	}

	if rev.OriginalFilename == "" {
		return fmt.Errorf("deleting revisions for this file type is not supported")
	}

	err = g.service.Revisions.Delete(args.FileId, args.RevisionId).Do()
	if err != nil {
		return fmt.Errorf("failed to delete revision:%v", err)
	}

	if args.JsonOut {
		if jb, err := json.Marshal(map[string]string{
			"revId": args.RevisionId,
			"mesg":  "deleted",
		}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	_, _ = fmt.Fprintf(args.Out, "Deleted revision '%s'\n", args.RevisionId)
	return nil
}
