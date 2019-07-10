package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
)

var DefaultExportMime = map[string]string{
	"application/vnd.google-apps.form":         "application/zip",
	"application/vnd.google-apps.document":     "application/pdf",
	"application/vnd.google-apps.drawing":      "image/svg+xml",
	"application/vnd.google-apps.spreadsheet":  "text/csv",
	"application/vnd.google-apps.script":       "application/vnd.google-apps.script+json",
	"application/vnd.google-apps.presentation": "application/pdf",
}

type ExportArgs struct {
	Out        io.Writer
	Id         string
	PrintMimes bool
	Mime       string
	Force      bool
	JsonOut    bool
}

func (g *Drive) Export(args ExportArgs) error {
	f, err := g.service.Files.Get(args.Id).Fields("name", "mimeType").Do()
	if err != nil {
		return fmt.Errorf("failed to get file: %s", err)
	}

	if args.PrintMimes {
		return g.printMimes(args, f.MimeType)
	}

	exportMime, err := getExportMime(args.Mime, f.MimeType)
	if err != nil {
		return err
	}

	filename := getExportFilename(f.Name, exportMime)

	res, err := g.service.Files.Export(args.Id, exportMime).Download()
	if err != nil {
		return fmt.Errorf("failed to download file: %s", err)
	}

	// Close body on function exit
	defer res.Body.Close()

	// Check if file exists
	if !args.Force && fileExists(filename) {
		return fmt.Errorf("file '%s' already exists, use --force to overwrite", filename)
	}

	// Create new file
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create new file '%s': %s", filename, err)
	}

	// Close file on function exit
	defer outFile.Close()

	// Save file to disk
	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		return fmt.Errorf("failed saving file: %s", err)
	}

	if args.JsonOut {
		if jb, err := json.Marshal(map[string]string{
			"exported": filename,
			"mimeType": exportMime,
		}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	} else {
		_, _ = fmt.Fprintf(args.Out, "Exported '%s' with mime type: '%s'\n", filename, exportMime)
	}

	return nil
}

func (g *Drive) printMimes(args ExportArgs, mimeType string) error {
	about, err := g.service.About.Get().Fields("exportFormats").Do()
	if err != nil {
		return fmt.Errorf("failed to get about: %s", err)
	}

	mimes, ok := about.ExportFormats[mimeType]
	if !ok {
		return fmt.Errorf("file with type '%s' cannot be exported", mimeType)
	}

	if args.JsonOut {
		if jb, err := json.Marshal(map[string][]string{
			"mimeTypes": mimes,
		}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	} else {
		_, _ = fmt.Fprintf(args.Out, "Available mime types: %s\n", formatList(mimes))
	}
	return nil
}

func getExportMime(userMime, fileMime string) (string, error) {
	if userMime != "" {
		return userMime, nil
	}

	defaultMime, ok := DefaultExportMime[fileMime]
	if !ok {
		return "", fmt.Errorf("file with type '%s' does not have a default export mime, and can probably not be exported", fileMime)
	}

	return defaultMime, nil
}

func getExportFilename(name, mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		return name
	}

	return name + extensions[0]
}
