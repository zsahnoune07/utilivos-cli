package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/utilivos/utilivos-cli/internal/update"
)

var defaultSeed = "#0F766E"

var defaultLight = map[string]string{
	"background":  "#F8FAFC",
	"surface":     "#EEFCF9",
	"primary":     "#00534D",
	"onPrimary":   "#FFFFFF",
	"secondary":   "#215055",
	"onSecondary": "#FFFFFF",
	"text":        "#111E1C",
	"muted":       "#394A48",
	"error":       "#98000A",
	"onError":     "#FFFFFF",
	"success":     "#1A7F37",
	"warning":     "#B05A00",
	"surfaceAlt":  "#F1F5F9",
	"outline":     "#576865",
}

var defaultDark = map[string]string{
	"background":  "#0E1416",
	"surface":     "#152025",
	"primary":     "#00DECF",
	"onPrimary":   "#002A26",
	"secondary":   "#A1CED4",
	"onSecondary": "#00292D",
	"text":        "#E2F1EE",
	"muted":       "#B8CAC7",
	"error":       "#FFB4AB",
	"onError":     "#520003",
	"success":     "#7EE787",
	"warning":     "#FFB86B",
	"surfaceAlt":  "#132126",
	"outline":     "#8C9F9B",
}

var updatedLight = map[string]string{
	"background":  "#F1F5F9",
	"surface":     "#E2E8F0",
	"primary":     "#112233",
	"onPrimary":   "#FFFFFF",
	"secondary":   "#224466",
	"onSecondary": "#FFFFFF",
	"text":        "#0B0F14",
	"muted":       "#334155",
	"error":       "#B91C1C",
	"onError":     "#FFFFFF",
	"success":     "#047857",
	"warning":     "#B45309",
	"surfaceAlt":  "#CBD5E1",
	"outline":     "#64748B",
}

var updatedDark = map[string]string{
	"background":  "#020617",
	"surface":     "#0F172A",
	"primary":     "#334455",
	"onPrimary":   "#E2E8F0",
	"secondary":   "#446688",
	"onSecondary": "#E2E8F0",
	"text":        "#E2E8F0",
	"muted":       "#94A3B8",
	"error":       "#FCA5A5",
	"onError":     "#450A0A",
	"success":     "#6EE7B7",
	"warning":     "#FDBA74",
	"surfaceAlt":  "#111827",
	"outline":     "#94A3B8",
}

type fixtureOptions struct {
	omitWeb    bool
	omitMobile bool
	omitSDK    bool
}

func TestExportGeneratesConfigFromCanonicalMobile(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := filepath.Join(root, "utilivos.theme.json")

	result, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("run theme sync export: %v", err)
	}
	if result.Mode != ModeExport {
		t.Fatalf("expected export mode, got %s", result.Mode)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load generated config: %v", err)
	}
	if cfg.SeedColor != defaultSeed {
		t.Fatalf("expected seed %s, got %s", defaultSeed, cfg.SeedColor)
	}
	if cfg.Colors.Light.Primary != defaultLight["primary"] {
		t.Fatalf("unexpected light primary %s", cfg.Colors.Light.Primary)
	}
}

func TestApplyUpdatesAllProjects(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := writeThemeConfig(t, root, "#123456", updatedLight, updatedDark)

	result, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("run theme sync apply: %v", err)
	}
	if result.Mode != ModeApply {
		t.Fatalf("expected apply mode, got %s", result.Mode)
	}
	if changedCount(result.Changes) == 0 {
		t.Fatal("expected file changes during apply")
	}

	mobilePath := filepath.Join(root, "utilivos-mobile", "lib", "core", "theme", "app_design_tokens.dart")
	mobileContent := readText(t, mobilePath)
	if !strings.Contains(mobileContent, "seedColor = Color(0xFF123456)") {
		t.Fatal("mobile seedColor was not updated")
	}
	if !strings.Contains(mobileContent, "primary: Color(0xFF112233)") {
		t.Fatal("mobile light primary was not updated")
	}
	if !strings.Contains(mobileContent, "primary: Color(0xFF334455)") {
		t.Fatal("mobile dark primary was not updated")
	}

	webPath := filepath.Join(root, "utilivos-web", "app", "globals.css")
	webContent := readText(t, webPath)
	if !strings.Contains(webContent, webThemeStartMarker) {
		t.Fatal("web managed theme block was not inserted")
	}
	if !strings.Contains(webContent, "--uv-web-primary: #112233;") {
		t.Fatal("web light primary not updated")
	}
	if !strings.Contains(webContent, `:root[data-uv-theme="dark"]`) {
		t.Fatal("web dark mode block missing")
	}

	tokensPath := filepath.Join(root, "docs", "design-system", "tokens.json")
	tokensContent := readText(t, tokensPath)
	if !strings.Contains(tokensContent, `"seed_color": "#123456"`) {
		t.Fatal("sdk token seed_color not updated")
	}
	if !strings.Contains(tokensContent, `"primary": "#112233"`) {
		t.Fatal("sdk token light primary not updated")
	}
}

func TestExportReportsMismatches(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := filepath.Join(root, "utilivos.theme.json")

	result, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath, DryRun: true})
	if err != nil {
		t.Fatalf("run export dry-run: %v", err)
	}
	if len(result.Mismatches) == 0 {
		t.Fatal("expected mismatches due legacy web theme")
	}
	foundWebDarkMissing := false
	for _, mismatch := range result.Mismatches {
		if mismatch.Project == webProjectName && mismatch.Key == "dark.background" && mismatch.Note == "missing" {
			foundWebDarkMissing = true
			break
		}
	}
	if !foundWebDarkMissing {
		t.Fatal("expected missing dark.background mismatch for web")
	}
}

func TestDryRunDoesNotWriteFiles(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := writeThemeConfig(t, root, "#123456", updatedLight, updatedDark)
	mobilePath := filepath.Join(root, "utilivos-mobile", "lib", "core", "theme", "app_design_tokens.dart")
	before := readText(t, mobilePath)

	result, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath, DryRun: true})
	if err != nil {
		t.Fatalf("run apply dry-run: %v", err)
	}
	if result.Mode != ModeApply {
		t.Fatalf("expected apply mode, got %s", result.Mode)
	}
	after := readText(t, mobilePath)
	if before != after {
		t.Fatal("dry-run should not modify files")
	}
	if changedCount(result.Changes) == 0 {
		t.Fatal("expected pending changes in dry-run")
	}
}

func TestMissingProjectHandling(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{omitWeb: true})
	configPath := writeThemeConfig(t, root, "#123456", updatedLight, updatedDark)

	if _, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath}); err == nil {
		t.Fatal("expected failure when web project is missing without --allow-missing")
	}

	result, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath, AllowMissing: true})
	if err != nil {
		t.Fatalf("run apply with allow-missing: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for missing web project")
	}
}

func TestSDKCSSPropagation(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := writeThemeConfig(t, root, "#123456", updatedLight, updatedDark)

	if _, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath}); err != nil {
		t.Fatalf("run apply: %v", err)
	}

	for _, path := range sdkCSSPaths(root) {
		content := readText(t, path)
		if !strings.Contains(content, "--uv-color-primary: #112233;") {
			t.Fatalf("light primary missing in %s", path)
		}
		if !strings.Contains(content, "--uv-color-primary: #334455;") {
			t.Fatalf("dark primary missing in %s", path)
		}
	}
}

func TestIdempotentRerunProducesNoChanges(t *testing.T) {
	root := createWorkspaceFixture(t, fixtureOptions{})
	configPath := writeThemeConfig(t, root, "#123456", updatedLight, updatedDark)

	if _, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath}); err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	second, err := RunThemeSync(SyncOptions{WorkspaceRoot: root, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if changedCount(second.Changes) != 0 {
		t.Fatalf("expected no changes on second apply, got %d", changedCount(second.Changes))
	}
}

func createWorkspaceFixture(t *testing.T, options fixtureOptions) string {
	t.Helper()
	root := t.TempDir()

	if !options.omitMobile {
		mobilePath := filepath.Join(root, "utilivos-mobile", "lib", "core", "theme", "app_design_tokens.dart")
		writeText(t, mobilePath, buildMobileTokens(defaultSeed, defaultLight, defaultDark))
	}

	if !options.omitWeb {
		webPath := filepath.Join(root, "utilivos-web", "app", "globals.css")
		writeText(t, webPath, buildLegacyWebCSS())
	}

	if !options.omitSDK {
		tokensPath := filepath.Join(root, "docs", "design-system", "tokens.json")
		writeText(t, tokensPath, buildSDKTokensJSON(defaultSeed, defaultLight, defaultDark))

		baseCSS := buildSDKBaseCSS(defaultLight, defaultDark)
		for _, path := range sdkCSSPaths(root) {
			writeText(t, path, baseCSS)
		}
	}

	return root
}

func sdkCSSPaths(root string) []string {
	return []string{
		filepath.Join(root, "utilivos-sdk", "ui", "base.css"),
		filepath.Join(root, "utilivos-sdk", "internal", "templates", "assets", "base.css"),
		filepath.Join(root, "utilivos-sdk", "examples", "hello", "tool", "utilivos", "base.css"),
		filepath.Join(root, "utilivos-sdk", "tests", "fixtures", "sample", "tool", "utilivos", "base.css"),
	}
}

func writeThemeConfig(t *testing.T, root, seed string, light, dark map[string]string) string {
	t.Helper()
	cfg, err := NewConfigFromCanonical(seed, light, dark)
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	encoded, err := EncodeConfig(cfg)
	if err != nil {
		t.Fatalf("encode config: %v", err)
	}
	path := filepath.Join(root, "utilivos.theme.json")
	writeText(t, path, string(encoded))
	return path
}

func changedCount(changes []update.FileChange) int {
	count := 0
	for _, change := range changes {
		if change.Before != change.After {
			count++
		}
	}
	return count
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func buildMobileTokens(seed string, light, dark map[string]string) string {
	return "import 'package:flutter/material.dart';\n\n" +
		"class AppDesignTokens {\n" +
		"  const AppDesignTokens._();\n\n" +
		"  static const Color seedColor = Color(0xFF" + strings.TrimPrefix(strings.ToUpper(seed), "#") + ");\n\n" +
		"  static const AppColorPalette light = AppColorPalette(\n" +
		"    mode: 'light',\n" +
		buildMobilePaletteRows(light) +
		"  );\n\n" +
		"  static const AppColorPalette dark = AppColorPalette(\n" +
		"    mode: 'dark',\n" +
		buildMobilePaletteRows(dark) +
		"  );\n" +
		"}\n\n" +
		"class AppColorPalette {\n" +
		"  const AppColorPalette({\n" +
		"    required this.mode,\n" +
		"    required this.background,\n" +
		"    required this.surface,\n" +
		"    required this.primary,\n" +
		"    required this.onPrimary,\n" +
		"    required this.secondary,\n" +
		"    required this.onSecondary,\n" +
		"    required this.text,\n" +
		"    required this.muted,\n" +
		"    required this.error,\n" +
		"    required this.onError,\n" +
		"    required this.success,\n" +
		"    required this.warning,\n" +
		"    required this.surfaceAlt,\n" +
		"    required this.outline,\n" +
		"  });\n" +
		"\n" +
		"  final String mode;\n" +
		"  final Color background;\n" +
		"  final Color surface;\n" +
		"  final Color primary;\n" +
		"  final Color onPrimary;\n" +
		"  final Color secondary;\n" +
		"  final Color onSecondary;\n" +
		"  final Color text;\n" +
		"  final Color muted;\n" +
		"  final Color error;\n" +
		"  final Color onError;\n" +
		"  final Color success;\n" +
		"  final Color warning;\n" +
		"  final Color surfaceAlt;\n" +
		"  final Color outline;\n" +
		"}\n"
}

func buildMobilePaletteRows(palette map[string]string) string {
	builder := strings.Builder{}
	for _, role := range RoleOrder {
		builder.WriteString("    " + role + ": Color(0xFF" + strings.TrimPrefix(strings.ToUpper(palette[role]), "#") + "),\n")
	}
	return builder.String()
}

func buildSDKTokensJSON(seed string, light, dark map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString("{\n")
	builder.WriteString("  \"design_system_version\": 1,\n")
	builder.WriteString("  \"source\": {\n")
	builder.WriteString("    \"seed_color\": \"" + strings.ToUpper(seed) + "\"\n")
	builder.WriteString("  },\n")
	builder.WriteString("  \"colors\": {\n")
	builder.WriteString("    \"light\": {\n")
	builder.WriteString(buildJSONPaletteRows(light))
	builder.WriteString("    },\n")
	builder.WriteString("    \"dark\": {\n")
	builder.WriteString(buildJSONPaletteRows(dark))
	builder.WriteString("    }\n")
	builder.WriteString("  }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func buildJSONPaletteRows(palette map[string]string) string {
	builder := strings.Builder{}
	for index, role := range RoleOrder {
		suffix := ","
		if index == len(RoleOrder)-1 {
			suffix = ""
		}
		builder.WriteString("      \"" + role + "\": \"" + strings.ToUpper(palette[role]) + "\"" + suffix + "\n")
	}
	return builder.String()
}

func buildSDKBaseCSS(light, dark map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString(":root {\n  --uv-space-1: 4px;\n}\n\n")
	builder.WriteString(serializeSDKColorBlock(":root", light))
	builder.WriteString("\n\n")
	builder.WriteString(serializeSDKColorBlock(`:root[data-uv-theme="dark"]`, dark))
	builder.WriteString("\n\n")
	builder.WriteString("@media (prefers-color-scheme: dark) {\n")
	inner := serializeSDKColorBlock(`:root:not([data-uv-theme="light"])`, dark)
	for _, line := range strings.Split(inner, "\n") {
		if line == "" {
			continue
		}
		builder.WriteString("  " + line + "\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func serializeSDKColorBlock(selector string, palette map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString(selector + " {\n")
	for _, role := range RoleOrder {
		builder.WriteString("  --uv-color-" + toKebab(role) + ": " + strings.ToLower(palette[role]) + ";\n")
	}
	builder.WriteString("}\n")
	return strings.TrimRight(builder.String(), "\n")
}

func buildLegacyWebCSS() string {
	return `:root {
  --bg: #F9FBFF;
  --panel: #FFFFFF;
  --text: #0F172A;
  --muted: #475569;
  --primary: #0F766E;
  --border: #DBE4EE;
}

html,
body {
  margin: 0;
  background:
    radial-gradient(circle at top right, rgba(15, 118, 110, 0.1), transparent 45%),
    radial-gradient(circle at 15% 20%, rgba(56, 189, 248, 0.1), transparent 35%),
    var(--bg);
}
`
}
