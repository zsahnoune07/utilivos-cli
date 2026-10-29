package publish

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/utilivos/utilivos-cli/internal/command"
	"golang.org/x/term"
)

// NewPublishCommand registers the publish subcommand.
func NewPublishCommand() *command.Node {
	return &command.Node{
		Name:    "publish",
		Summary: "Publish a .ubb bundle to the Utilivos admin API",
		Description: "Publish reads manifest from the bundle, uploads to POST /v1/admin/plugins/publish.\n\n" +
			"Loads .env from the current directory if present (does not override existing env vars).\n" +
			"Primary env: UTILIVOS_API_BASE_URL, UTILIVOS_ADMIN_API_KEY (legacy: UTILIVO_API_URL, UTILIVO_ADMIN_API_KEY).\n\n" +
			"Interactive by default when stdin is a TTY: choose category, plugin/version status, optional changelog.\n" +
			"Use --non-interactive for CI (requires category flags if manifest category is unknown to the API).\n\n" +
			"Usage: utilivos-cli publish <file.ubb> [flags]\n\n" +
			"Flags: --env-file, --url, --api-key, --non-interactive, --category-slug, --category-name,\n" +
			"       --plugin-status, --version-status, --changelog, --timeout, --json",
		Run: runPublish,
	}
}

func runPublish(ctx *command.Context, args []string) (int, error) {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	envFile := fs.String("env-file", "", "Path to .env file (default: .env in cwd if present)")
	nonInteractive := fs.Bool("non-interactive", false, "No prompts; use flags/env only")
	urlFlag := fs.String("url", "", "API base URL (overrides env)")
	apiKeyFlag := fs.String("api-key", "", "Admin API key (overrides env)")
	categorySlugFlag := fs.String("category-slug", "", "Category slug when manifest category is not in API (non-interactive)")
	categoryNameFlag := fs.String("category-name", "", "Category display name for new categories (non-interactive)")
	pluginStatusFlag := fs.String("plugin-status", "", "draft|published|archived (non-interactive; default published)")
	versionStatusFlag := fs.String("version-status", "", "draft|published|deprecated (non-interactive; default published)")
	changelog := fs.String("changelog", "", "Optional changelog text")
	jsonOut := fs.Bool("json", false, "Print JSON response (message, plugin, version, category)")
	timeout := fs.Duration("timeout", 5*time.Minute, "HTTP timeout")

	ubbPath, parseArgs := splitLeadingPositional(args)
	if err := fs.Parse(parseArgs); err != nil {
		return command.ExitUsage, err
	}
	if ubbPath == "" {
		if fs.NArg() != 1 {
			return command.ExitUsage, fmt.Errorf("usage: utilivos-cli publish <file.ubb> [flags]")
		}
		ubbPath = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return command.ExitUsage, fmt.Errorf("usage: utilivos-cli publish <file.ubb> [flags]")
	}

	if err := LoadEnvFile(ctx.Cwd, strings.TrimSpace(*envFile), true); err != nil {
		return command.ExitConfig, err
	}

	baseURL := strings.TrimSpace(*urlFlag)
	if baseURL == "" {
		baseURL = ResolveBaseURL()
	}
	apiKey := strings.TrimSpace(*apiKeyFlag)
	if apiKey == "" {
		apiKey = ResolveAPIKey()
	}
	if baseURL == "" {
		return command.ExitConfig, fmt.Errorf("API base URL is required (.env UTILIVOS_API_BASE_URL or UTILIVO_API_URL, or --url)")
	}
	if apiKey == "" {
		return command.ExitConfig, fmt.Errorf("admin API key is required (.env UTILIVOS_ADMIN_API_KEY or UTILIVO_ADMIN_API_KEY, or --api-key)")
	}

	m, err := ReadManifestFromUBB(ubbPath)
	if err != nil {
		return command.ExitConfig, err
	}
	if err := ValidateForPublish(m); err != nil {
		return command.ExitConfig, err
	}

	cats, err := FetchAdminCategories(context.Background(), baseURL, apiKey, *timeout)
	if err != nil {
		return command.ExitConfig, fmt.Errorf("list categories: %w", err)
	}

	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	useInteractive := !*nonInteractive && stdinTTY
	if !*nonInteractive && !stdinTTY {
		_, _ = fmt.Fprintln(ctx.Stderr, "stdin is not a terminal; running in non-interactive mode.")
		useInteractive = false
	}

	opts := Options{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Changelog: strings.TrimSpace(*changelog),
		Timeout:   *timeout,
	}

	if useInteractive {
		cs, cn, ps, vs, cl, err := RunInteractivePrompts(os.Stdin, os.Stdout, ctx.Stderr, m, cats)
		if err != nil {
			return command.ExitUsage, err
		}
		opts.CategorySlug, opts.CategoryName = cs, cn
		opts.PluginStatus, opts.VersionStatus = ps, vs
		if cl != "" {
			opts.Changelog = cl
		}
		summary, _ := json.MarshalIndent(map[string]string{
			"plugin":           Slugify(m.Slug),
			"version":          strings.TrimSpace(m.Version),
			"category_slug":    opts.CategorySlug,
			"category_name":    opts.CategoryName,
			"plugin_status":    opts.PluginStatus,
			"version_status": opts.VersionStatus,
			"pricing_type":     TierToPricing(m.Tier),
			"changelog":        opts.Changelog,
		}, "", "  ")
		_, _ = fmt.Fprintf(ctx.Stdout, "\nReady to publish:\n%s\n\n", string(summary))
		confirm := false
		if err := survey.AskOne(&survey.Confirm{Message: "Send publish request?", Default: true}, &confirm,
			survey.WithStdio(os.Stdin, os.Stdout, ctx.Stderr)); err != nil {
			return command.ExitUsage, err
		}
		if !confirm {
			_, _ = fmt.Fprintln(ctx.Stdout, "Aborted.")
			return command.ExitOK, nil
		}
	} else {
		cs, cn, err := ResolveCategoryNonInteractive(m, cats, *categorySlugFlag, *categoryNameFlag)
		if err != nil {
			return command.ExitConfig, err
		}
		opts.CategorySlug, opts.CategoryName = cs, cn
		if ps := strings.TrimSpace(*pluginStatusFlag); ps != "" {
			opts.PluginStatus = ps
		}
		if vs := strings.TrimSpace(*versionStatusFlag); vs != "" {
			opts.VersionStatus = vs
		}
	}

	res, err := Publish(context.Background(), ubbPath, m, opts)
	if err != nil {
		return command.ExitApply, err
	}

	if *jsonOut {
		out := struct {
			Message  string          `json:"message"`
			Plugin   json.RawMessage `json:"plugin"`
			Version  json.RawMessage `json:"version"`
			Category json.RawMessage `json:"category"`
		}{
			Message:  res.Message,
			Plugin:   res.Plugin,
			Version:  res.Version,
			Category: res.Category,
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return command.ExitApply, err
		}
		fmt.Fprintln(ctx.Stdout, string(b))
		return command.ExitOK, nil
	}

	fmt.Fprintln(ctx.Stdout, res.Message)
	return command.ExitOK, nil
}

func splitLeadingPositional(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}
