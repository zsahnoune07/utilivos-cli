package theme

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/utilivos/utilivos-cli/internal/update"
)

type Mode string

const (
	ModeExport Mode = "export"
	ModeApply  Mode = "apply"
)

type SyncOptions struct {
	WorkspaceRoot string
	ConfigPath    string
	ForceExport   bool
	ForceApply    bool
	DryRun        bool
	Diff          bool
	AllowMissing  bool
}

type Mismatch struct {
	Project  string
	Key      string
	Expected string
	Actual   string
	Note     string
}

type SyncResult struct {
	Mode       Mode
	ConfigPath string
	Warnings   []string
	Mismatches []Mismatch
	Plans      []ProjectPlan
	Changes    []update.FileChange
	Diffs      []ValueDiff
}

func RunThemeSync(opts SyncOptions) (SyncResult, error) {
	if opts.WorkspaceRoot == "" {
		return SyncResult{}, fmt.Errorf("workspace root is required")
	}
	if opts.ConfigPath == "" {
		return SyncResult{}, fmt.Errorf("config path is required")
	}
	if opts.ForceApply && opts.ForceExport {
		return SyncResult{}, fmt.Errorf("--apply and --export cannot be combined")
	}

	adapters := []Adapter{MobileAdapter{}, WebAdapter{}, SDKAdapter{}}
	active, warnings, err := resolveActiveAdapters(adapters, opts.WorkspaceRoot, opts.AllowMissing)
	if err != nil {
		return SyncResult{}, err
	}

	mode, err := resolveMode(opts.ConfigPath, opts.ForceExport, opts.ForceApply)
	if err != nil {
		return SyncResult{}, err
	}

	result := SyncResult{
		Mode:       mode,
		ConfigPath: opts.ConfigPath,
		Warnings:   warnings,
	}

	switch mode {
	case ModeExport:
		mobile, ok := active[mobileProjectName]
		if !ok {
			return SyncResult{}, fmt.Errorf("cannot export canonical theme: %s is missing", mobileProjectName)
		}
		canonical, err := mobile.Read(opts.WorkspaceRoot)
		if err != nil {
			return SyncResult{}, err
		}
		cfg, err := NewConfigFromCanonical(canonical.SeedColor, canonical.Light, canonical.Dark)
		if err != nil {
			return SyncResult{}, fmt.Errorf("build export config: %w", err)
		}
		encoded, err := EncodeConfig(cfg)
		if err != nil {
			return SyncResult{}, err
		}

		before := ""
		if data, readErr := os.ReadFile(opts.ConfigPath); readErr == nil {
			before = string(data)
		}
		result.Changes = append(result.Changes, update.FileChange{
			Path:        opts.ConfigPath,
			Before:      before,
			After:       string(encoded),
			Description: "export canonical theme config",
		})

		mismatches, mismatchWarnings, err := collectMismatches(active, opts.WorkspaceRoot, cfg)
		if err != nil {
			return SyncResult{}, err
		}
		result.Mismatches = mismatches
		result.Warnings = append(result.Warnings, mismatchWarnings...)

		if !opts.DryRun {
			if err := update.Apply(result.Changes); err != nil {
				return SyncResult{}, err
			}
		}
		return result, nil

	case ModeApply:
		cfg, err := LoadConfig(opts.ConfigPath)
		if err != nil {
			return SyncResult{}, err
		}
		plans := make([]ProjectPlan, 0, len(active))
		for _, project := range []string{mobileProjectName, webProjectName, sdkProjectName} {
			adapter, ok := active[project]
			if !ok {
				continue
			}
			plan, planErr := adapter.Plan(opts.WorkspaceRoot, cfg)
			if planErr != nil {
				return SyncResult{}, planErr
			}
			plans = append(plans, plan)
		}
		result.Plans = plans
		result.Changes = flattenChanges(plans)
		result.Diffs = flattenDiffs(plans)
		if !opts.DryRun {
			if err := update.Apply(result.Changes); err != nil {
				return SyncResult{}, err
			}
		}
		return result, nil
	default:
		return SyncResult{}, fmt.Errorf("unsupported mode %q", mode)
	}
}

func resolveMode(configPath string, forceExport, forceApply bool) (Mode, error) {
	if forceExport {
		return ModeExport, nil
	}
	if forceApply {
		if !exists(configPath) {
			return "", fmt.Errorf("config file does not exist: %s", configPath)
		}
		return ModeApply, nil
	}
	if exists(configPath) {
		return ModeApply, nil
	}
	return ModeExport, nil
}

func resolveActiveAdapters(adapters []Adapter, root string, allowMissing bool) (map[string]Adapter, []string, error) {
	active := make(map[string]Adapter)
	warnings := make([]string, 0)
	missingFatal := make([]string, 0)
	for _, adapter := range adapters {
		detection := adapter.Detect(root)
		if detection.Present {
			active[adapter.Name()] = adapter
			continue
		}
		message := fmt.Sprintf("%s missing required files: %s", adapter.Name(), strings.Join(detection.Missing, ", "))
		if allowMissing {
			warnings = append(warnings, message)
			continue
		}
		missingFatal = append(missingFatal, message)
	}
	if len(missingFatal) > 0 {
		sort.Strings(missingFatal)
		return nil, warnings, fmt.Errorf("missing required project targets:\n - %s", strings.Join(missingFatal, "\n - "))
	}
	return active, warnings, nil
}

func collectMismatches(active map[string]Adapter, root string, cfg ThemeConfig) ([]Mismatch, []string, error) {
	mismatches := make([]Mismatch, 0)
	warnings := make([]string, 0)
	canonicalLight := PaletteToMap(cfg.Colors.Light)
	canonicalDark := PaletteToMap(cfg.Colors.Dark)

	projects := make([]string, 0, len(active))
	for project := range active {
		projects = append(projects, project)
	}
	sort.Strings(projects)

	for _, project := range projects {
		if project == mobileProjectName {
			continue
		}
		observation, err := active[project].Read(root)
		if err != nil {
			return nil, nil, err
		}
		if observation.SeedColor != "" && strings.ToUpper(observation.SeedColor) != strings.ToUpper(cfg.SeedColor) {
			mismatches = append(mismatches, Mismatch{
				Project:  project,
				Key:      "seedColor",
				Expected: strings.ToUpper(cfg.SeedColor),
				Actual:   strings.ToUpper(observation.SeedColor),
			})
		}
		for _, role := range RoleOrder {
			expected := canonicalLight[role]
			actual, ok := observation.Light[role]
			if !ok {
				mismatches = append(mismatches, Mismatch{Project: project, Key: "light." + role, Expected: expected, Actual: "", Note: "missing"})
			} else if strings.ToUpper(actual) != strings.ToUpper(expected) {
				mismatches = append(mismatches, Mismatch{Project: project, Key: "light." + role, Expected: expected, Actual: strings.ToUpper(actual)})
			}
		}
		for _, role := range RoleOrder {
			expected := canonicalDark[role]
			actual, ok := observation.Dark[role]
			if !ok {
				mismatches = append(mismatches, Mismatch{Project: project, Key: "dark." + role, Expected: expected, Actual: "", Note: "missing"})
			} else if strings.ToUpper(actual) != strings.ToUpper(expected) {
				mismatches = append(mismatches, Mismatch{Project: project, Key: "dark." + role, Expected: expected, Actual: strings.ToUpper(actual)})
			}
		}
	}
	if len(mismatches) > 0 {
		warnings = append(warnings, fmt.Sprintf("detected %d cross-project theme mismatch(es); export used mobile canonical values", len(mismatches)))
	}
	return mismatches, warnings, nil
}

func flattenChanges(plans []ProjectPlan) []update.FileChange {
	changes := make([]update.FileChange, 0)
	for _, plan := range plans {
		changes = append(changes, plan.Changes...)
	}
	update.SortChanges(changes)
	return changes
}

func flattenDiffs(plans []ProjectPlan) []ValueDiff {
	diffs := make([]ValueDiff, 0)
	for _, plan := range plans {
		diffs = append(diffs, plan.Diffs...)
	}
	SortDiffs(diffs)
	return diffs
}
