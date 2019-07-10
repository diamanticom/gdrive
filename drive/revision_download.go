package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"
)

type DownloadRevisionArgs struct {
	Out        io.Writer
	Progress   io.Writer
	FileId     string
	RevisionId string
	Path       string
	Force      bool
	Stdout     bool
	Timeout    time.Duration
	JsonOut    bool
}

func (g *Drive) DownloadRevision(args DownloadRevisionArgs) (err error) {
	getRev := g.service.Revisions.Get(args.FileId, args.RevisionId)

	rev, err := getRev.Fields("originalFilename").Do()
	if err != nil {
		return fmt.Errorf("failed to get file: %s", err)
	}

	if rev.OriginalFilename == "" {
		return fmt.Errorf("download is not supported for this file type")
	}

	// Get timeout reader wrapper and context
	timeoutReaderWrapper, ctx := getTimeoutReaderWrapperContext(args.Timeout)

	res, err := getRev.Context(ctx).Download()
	if err != nil {
		if isTimeoutError(err) {
			return fmt.Errorf("failed to download file: timeout, no data was transferred for %v", args.Timeout)
		}
		return fmt.Errorf("failed to download file: %s", err)
	}

	// Close body on function exit
	defer res.Body.Close()

	// Discard other output if file is written to stdout
	out := args.Out
	if args.Stdout {
		out = ioutil.Discard
	}

	// Path to file
	fpath := filepath.Join(args.Path, rev.OriginalFilename)

	if !args.JsonOut {
		_, _ = fmt.Fprintf(out, "Downloading %s -> %s\n", rev.OriginalFilename, fpath)
	}

	bytes, rate, err := g.saveFile(saveFileArgs{
		out:           args.Out,
		body:          timeoutReaderWrapper(res.Body),
		contentLength: res.ContentLength,
		fpath:         fpath,
		force:         args.Force,
		stdout:        args.Stdout,
		progress:      args.Progress,
	})

	if err != nil {
		return err
	}

	if args.JsonOut {
		if jb, err := json.Marshal(map[string]string{}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
		}
		return nil
	}

	_, _ = fmt.Fprintf(out, "Download complete, rate: %s/s, total size: %s\n",
		formatSize(rate, false), formatSize(bytes, false))
	return nil
}
