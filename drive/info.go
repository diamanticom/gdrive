package drive

import (
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

type FileInfoArgs struct {
	Out         io.Writer
	Id          string
	SizeInBytes bool
	JsonOut     bool
}

func (g *Drive) Info(args FileInfoArgs) error {
	f, err := g.service.Files.Get(args.Id).Fields("id", "name", "size", "createdTime", "modifiedTime", "md5Checksum", "mimeType", "parents", "shared", "description", "webContentLink", "webViewLink").Do()
	if err != nil {
		return fmt.Errorf("failed to get file: %s", err)
	}

	pathfinder := g.newPathfinder()
	absPath, err := pathfinder.absPath(f)
	if err != nil {
		return err
	}

	if args.JsonOut {
		PrintFileInfoJson(PrintFileInfoArgs{
			Out:         args.Out,
			File:        f,
			Path:        absPath,
			SizeInBytes: args.SizeInBytes,
		})
	} else {
		PrintFileInfo(PrintFileInfoArgs{
			Out:         args.Out,
			File:        f,
			Path:        absPath,
			SizeInBytes: args.SizeInBytes,
		})
	}

	return nil
}

type PrintFileInfoArgs struct {
	Out         io.Writer
	File        *drive.File
	Path        string
	SizeInBytes bool
}

func PrintFileInfoJson(args PrintFileInfoArgs) {
	f := args.File

	items := map[string]string{
		"id":          f.Id,
		"name":        f.Name,
		"path":        args.Path,
		"description": f.Description,
		"mimeType":    f.MimeType,
		"size":        formatSize(f.Size, args.SizeInBytes),
		"createdTime": formatDatetime(f.CreatedTime),
		"modified":    formatDatetime(f.ModifiedTime),
		"md5Checksum": f.Md5Checksum,
		"shared":      formatBool(f.Shared),
		"parents":     formatList(f.Parents),
		"viewUrl":     f.WebViewLink,
		"downloadUrl": f.WebContentLink,
	}

	jb, _ := json.Marshal(items)
	_, _ = fmt.Fprintln(args.Out, string(jb))
}

func PrintFileInfo(args PrintFileInfoArgs) {
	f := args.File

	items := []kv{
		{"Id", f.Id},
		{"Name", f.Name},
		{"Path", args.Path},
		{"Description", f.Description},
		{"Mime", f.MimeType},
		{"Size", formatSize(f.Size, args.SizeInBytes)},
		{"Created", formatDatetime(f.CreatedTime)},
		{"Modified", formatDatetime(f.ModifiedTime)},
		{"Md5sum", f.Md5Checksum},
		{"Shared", formatBool(f.Shared)},
		{"Parents", formatList(f.Parents)},
		{"ViewUrl", f.WebViewLink},
		{"DownloadUrl", f.WebContentLink},
	}

	for _, item := range items {
		if item.value != "" {
			_, _ = fmt.Fprintf(args.Out, "%s: %s\n", item.key, item.value)
		}
	}
}
