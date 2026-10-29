package theme

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/utilivos/utilivos-cli/internal/command"
	"github.com/utilivos/utilivos-cli/internal/config"
	"github.com/utilivos/utilivos-cli/internal/workspace"
)

func NewThemeCommand() *command.Node {
	sync := &command.Node{
		Name:    "sync",
		Summary: "Export or apply cross-project theme colors",
		Run:     runThemeSync,
	}
	return &command.Node{
		Name:        "theme",
		Summary:     "Theme constants workflows",
		Description: "Manage cross-project theme constants and propagation.",
		Subcommands: map[string]*command.Node{"sync": sync},
	}
}

func runThemeSync(ctx *command.Context, args []string) (int, error) {
	fs := flag.NewFlagSet("theme sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		configPath   string
		workspaceArg string
		forceExport  bool
		forceApply   bool
		dryRun       bool
		diff         bool
		allowMissing bool
	)
	fs.StringVar(&configPath, "config", "", "Theme config file path (relative to workspace root)")
	fs.StringVar(&workspaceArg, "workspace", "", "Workspace directory")
	fs.BoolVar(&forceExport, "export", false, "Force export mode")
	fs.BoolVar(&forceApply, "apply", false, "Force apply mode")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")
	fs.BoolVar(&diff, "diff", false, "Show value-level diff summary")
	fs.BoolVar(&allowMissing, "allow-missing", false, "Skip missing target projects")

	if err := fs.Parse(args); err != nil {
		return command.ExitUsage, err
	}
	if fs.NArg() > 0 {
		return command.ExitUsage, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	startPath := ctx.Cwd
	if workspaceArg != "" {
		startPath = workspaceArg
	}
	ws, err := workspace.Discover(startPath)
	if err != nil {
		return command.ExitConfig, err
	}
	settings, err := config.LoadSettings(ws.Root)
	if err != nil {
		return command.ExitConfig, err
	}
	if configPath == "" {
		configPath = settings.DefaultThemeConfigPath
	}
	resolvedConfigPath := ResolveConfigPath(ws.Root, configPath)

	ctx.Logger.Infof("workspace: %s", ws.Root)
	ctx.Logger.Infof("config: %s", resolvedConfigPath)

	result, err := RunThemeSync(SyncOptions{
		WorkspaceRoot: ws.Root,
		ConfigPath:    resolvedConfigPath,
		ForceExport:   forceExport,
		ForceApply:    forceApply,
		DryRun:        dryRun,
		Diff:          diff,
		AllowMissing:  allowMissing,
	})
	if err != nil {
		return command.ExitApply, err
	}

	printSyncSummary(ctx, result, dryRun, diff)
	return command.ExitOK, nil
}

func printSyncSummary(ctx *command.Context, result SyncResult, dryRun, showDiff bool) {
	fmt.Fprintf(ctx.Stdout, "mode: %s\n", result.Mode)
	fmt.Fprintf(ctx.Stdout, "config: %s\n", result.ConfigPath)

	for _, warning := range result.Warnings {
		fmt.Fprintf(ctx.Stdout, "warning: %s\n", warning)
	}

	if len(result.Mismatches) > 0 {
		fmt.Fprintf(ctx.Stdout, "mismatches: %d\n", len(result.Mismatches))
		for _, mismatch := range result.Mismatches {
			if mismatch.Note != "" {
				fmt.Fprintf(ctx.Stdout, "  - [%s] %s expected=%s actual=%s (%s)\n", mismatch.Project, mismatch.Key, mismatch.Expected, blankAsDash(mismatch.Actual), mismatch.Note)
				continue
			}
			fmt.Fprintf(ctx.Stdout, "  - [%s] %s expected=%s actual=%s\n", mismatch.Project, mismatch.Key, mismatch.Expected, blankAsDash(mismatch.Actual))
		}
	}

	changed := 0
	for _, change := range result.Changes {
		if change.Before != change.After {
			changed++
		}
	}

	if dryRun {
		fmt.Fprintf(ctx.Stdout, "dry-run: %d file(s) would change\n", changed)
	} else {
		fmt.Fprintf(ctx.Stdout, "changed files: %d\n", changed)
	}

	if changed > 0 {
		for _, change := range result.Changes {
			if change.Before == change.After {
				continue
			}
			rel := change.Path
			if resultPath, err := filepath.Rel(ctx.Cwd, change.Path); err == nil {
				rel = resultPath
			}
			fmt.Fprintf(ctx.Stdout, "  - %s\n", rel)
		}
	}

	if showDiff {
		diffs := result.Diffs
		if len(diffs) == 0 && len(result.Mismatches) == 0 {
			fmt.Fprintln(ctx.Stdout, "diff: no value changes")
			return
		}
		if len(diffs) > 0 {
			fmt.Fprintln(ctx.Stdout, "value diff:")
			for _, diff := range diffs {
				fmt.Fprintf(ctx.Stdout, "  - [%s] %s: %s -> %s\n", diff.Project, diff.Key, blankAsDash(diff.Before), blankAsDash(diff.After))
			}
		}
	}
}

func blankAsDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
