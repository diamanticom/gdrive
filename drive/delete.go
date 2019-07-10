package drive

import (
	"encoding/json"
	"fmt"
	"io"
)

type DeleteArgs struct {
	Out       io.Writer
	Id        string
	Recursive bool
	JsonOut   bool
}

func (g *Drive) Delete(args DeleteArgs) error {
	f, err := g.service.Files.Get(args.Id).Fields("name", "mimeType").Do()
	if err != nil {
		return fmt.Errorf("failed to get file: %s", err)
	}

	if isDir(f) && !args.Recursive {
		return fmt.Errorf("'%s' is a directory, use the 'recursive' flag to delete directories", f.Name)
	}

	err = g.service.Files.Delete(args.Id).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file: %s", err)
	}

	mesg := fmt.Sprintf("Deleted '%s'\n", f.Name)

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

func (g *Drive) deleteFile(fileId string) error {
	err := g.service.Files.Delete(fileId).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file: %s", err)
	}
	return nil
}
