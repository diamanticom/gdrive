package drive

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
	"strings"
)

type ImportArgs struct {
	Out      io.Writer
	Mime     string
	Progress io.Writer
	Path     string
	Parents  []string
	JsonOut  bool
}

func (g *Drive) Import(args ImportArgs) error {
	fromMime := args.Mime
	if fromMime == "" {
		fromMime = getMimeType(args.Path)
	}
	if fromMime == "" {
		return fmt.Errorf("could not determine mime type of file, use --mime")
	}

	about, err := g.service.About.Get().Fields("importFormats").Do()
	if err != nil {
		return fmt.Errorf("failed to get about: %s", err)
	}

	toMimes, ok := about.ImportFormats[fromMime]
	if !ok || len(toMimes) == 0 {
		return fmt.Errorf("mime type '%s' is not supported for import", fromMime)
	}

	f, _, err := g.uploadFile(UploadArgs{
		Out:      ioutil.Discard,
		Progress: args.Progress,
		Path:     args.Path,
		Parents:  args.Parents,
		Mime:     toMimes[0],
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(args.Out, "Imported %s with mime type: '%s'\n", f.Id, toMimes[0])
	return nil
}

func getMimeType(path string) string {
	t := mime.TypeByExtension(filepath.Ext(path))
	return strings.Split(t, ";")[0]
}
