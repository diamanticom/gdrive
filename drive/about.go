package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

type AboutArgs struct {
	Out         io.Writer
	SizeInBytes bool
	JsonOut     bool
}

func (g *Drive) About(args AboutArgs) error {
	about, err := g.service.About.Get().Fields("maxImportSizes", "maxUploadSize", "storageQuota", "user").Do()
	if err != nil {
		return fmt.Errorf("failed to get about: %s", err)
	}

	user := about.User
	quota := about.StorageQuota

	if args.JsonOut {
		if jb, err := json.Marshal(about); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	_, _ = fmt.Fprintf(args.Out, "User: %s, %s\n", user.DisplayName, user.EmailAddress)
	_, _ = fmt.Fprintf(args.Out, "Used: %s\n", formatSize(quota.Usage, args.SizeInBytes))
	_, _ = fmt.Fprintf(args.Out, "Free: %s\n", formatSize(quota.Limit-quota.Usage, args.SizeInBytes))
	_, _ = fmt.Fprintf(args.Out, "Total: %s\n", formatSize(quota.Limit, args.SizeInBytes))
	_, _ = fmt.Fprintf(args.Out, "Max upload size: %s\n", formatSize(about.MaxUploadSize, args.SizeInBytes))

	return nil
}

type AboutImportArgs struct {
	Out io.Writer
}

func (g *Drive) AboutImport(args AboutImportArgs) (err error) {
	about, err := g.service.About.Get().Fields("importFormats").Do()
	if err != nil {
		return fmt.Errorf("failed to get about: %s", err)
	}
	printAboutFormats(args.Out, about.ImportFormats)
	return
}

type AboutExportArgs struct {
	Out io.Writer
}

func (g *Drive) AboutExport(args AboutExportArgs) (err error) {
	about, err := g.service.About.Get().Fields("exportFormats").Do()
	if err != nil {
		return fmt.Errorf("failed to get about: %s", err)
	}
	printAboutFormats(args.Out, about.ExportFormats)
	return
}

func printAboutFormats(out io.Writer, formats map[string][]string) {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 0, 3, ' ', 0)

	_, _ = fmt.Fprintln(w, "From\tTo")

	for from, toFormats := range formats {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", from, formatList(toFormats))
	}

	_ = w.Flush()
}
