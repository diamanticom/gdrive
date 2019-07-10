package drive

import (
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

const DirectoryMimeType = "application/vnd.google-apps.folder"

type MkdirArgs struct {
	Out         io.Writer
	Name        string
	Description string
	Parents     []string
	JsonOut     bool
}

func (g *Drive) Mkdir(args MkdirArgs) error {
	f, err := g.mkdir(args)
	if err != nil {
		return err
	}

	if args.JsonOut {
		if jb, err := json.Marshal(map[string]string{
			"id":   f.Id,
			"mesg": "directory created",
		}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	_, _ = fmt.Fprintf(args.Out, "Directory %s created\n", f.Id)
	return nil
}

func (g *Drive) mkdir(args MkdirArgs) (*drive.File, error) {
	dstFile := &drive.File{
		Name:        args.Name,
		Description: args.Description,
		MimeType:    DirectoryMimeType,
	}

	// Set parent folders
	dstFile.Parents = args.Parents

	// Create directory
	f, err := g.service.Files.Create(dstFile).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %s", err)
	}

	return f, nil
}
