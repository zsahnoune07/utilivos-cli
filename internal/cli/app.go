package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/utilivos/utilivos-cli/internal/command"
	"github.com/utilivos/utilivos-cli/internal/logutil"
	"github.com/utilivos/utilivos-cli/internal/publish"
	"github.com/utilivos/utilivos-cli/internal/theme"
)

func Run(argv []string, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		return command.ExitUsage, fmt.Errorf("argv is required")
	}
	args := argv[1:]
	verbose := false
	for len(args) > 0 {
		if args[0] == "--verbose" {
			verbose = true
			args = args[1:]
			continue
		}
		break
	}

	cwd, err := os.Getwd()
	if err != nil {
		return command.ExitConfig, fmt.Errorf("get working directory: %w", err)
	}
	ctx := &command.Context{
		Stdout: stdout,
		Stderr: stderr,
		Logger: logutil.New(stdout, verbose),
		Cwd:    cwd,
	}

	root := &command.Node{
		Name:    "utilivos-cli",
		Summary: "Utilivos internal cross-project workflow CLI",
		Description: "utilivos-cli\n\n" +
			"Usage: utilivos-cli [--verbose] <command> [subcommand] [flags]",
		Subcommands: map[string]*command.Node{
			"publish": publish.NewPublishCommand(),
			"theme":   theme.NewThemeCommand(),
		},
	}

	if len(args) == 0 {
		return root.Execute(ctx, args)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return root.Execute(ctx, []string{"help"})
	}
	return root.Execute(ctx, args)
}
