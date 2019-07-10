package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/gdrive-org/gdrive/utils"

	"github.com/gdrive-org/gdrive/cli"
)

func printVersion(ctx cli.Context) {
	fmt.Printf("%s: %s\n", utils.Name, utils.Version)
	fmt.Printf("Golang: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func printHelp(ctx cli.Context) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 3, ' ', 0)

	_, _ = fmt.Fprintf(w, "%s usage:\n\n", utils.Name)

	for _, h := range ctx.Handlers() {
		_, _ = fmt.Fprintf(w, "%s %s\t%s\n", utils.Name, h.Pattern, h.Description)
	}

	_ = w.Flush()
}

func printCommandHelp(ctx cli.Context) {
	args := ctx.Args()
	printCommandPrefixHelp(ctx, args.String("command"))
}

func printSubCommandHelp(ctx cli.Context) {
	args := ctx.Args()
	printCommandPrefixHelp(ctx, args.String("command"), args.String("subcommand"))
}

func printCommandPrefixHelp(ctx cli.Context, prefix ...string) {
	handler := getHandler(ctx.Handlers(), prefix)

	if handler == nil {
		utils.ExitF("Command not found")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 3, ' ', 0)

	_, _ = fmt.Fprintf(w, "%s\n", handler.Description)
	_, _ = fmt.Fprintf(w, "%s %s\n", utils.Name, handler.Pattern)
	for _, group := range handler.FlagGroups {
		_, _ = fmt.Fprintf(w, "\n%s:\n", group.Name)
		for _, flag := range group.Flags {
			boolFlag, isBool := flag.(cli.BoolFlag)
			if isBool && boolFlag.OmitValue {
				_, _ = fmt.Fprintf(w, "  %s\t%s\n", strings.Join(flag.GetPatterns(), ", "), flag.GetDescription())
			} else {
				_, _ = fmt.Fprintf(w, "  %s <%s>\t%s\n", strings.Join(flag.GetPatterns(), ", "), flag.GetName(), flag.GetDescription())
			}
		}
	}

	_ = w.Flush()
}

func getHandler(handlers []*cli.Handler, prefix []string) *cli.Handler {
	for _, h := range handlers {
		pattern := stripOptionals(h.SplitPattern())

		if len(prefix) > len(pattern) {
			continue
		}

		if utils.Equal(prefix, pattern[:len(prefix)]) {
			return h
		}
	}

	return nil
}

// Strip optional groups (<...>) from pattern
func stripOptionals(pattern []string) []string {
	var newArgs []string

	for _, arg := range pattern {
		if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
			continue
		}
		newArgs = append(newArgs, arg)
	}
	return newArgs
}
