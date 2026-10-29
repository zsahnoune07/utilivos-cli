package theme

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const webProjectName = "utilivos-web"

const (
	webThemeStartMarker = "/* utilivos-cli theme:start */"
	webThemeEndMarker   = "/* utilivos-cli theme:end */"
)

type WebAdapter struct{}

func (a WebAdapter) Name() string {
	return webProjectName
}

func (a WebAdapter) Detect(root string) Detection {
	path := a.globalsPath(root)
	missing := make([]string, 0)
	if !exists(path) {
		missing = append(missing, path)
	}
	return Detection{Project: a.Name(), Present: len(missing) == 0, Missing: missing}
}

func (a WebAdapter) Read(root string) (ThemeObservation, error) {
	path := a.globalsPath(root)
	content, err := readFile(path)
	if err != nil {
		return ThemeObservation{}, err
	}
	if strings.Contains(content, webThemeStartMarker) && strings.Contains(content, webThemeEndMarker) {
		light, dark, parseErr := parseManagedWebTheme(content)
		if parseErr != nil {
			return ThemeObservation{}, parseErr
		}
		return ThemeObservation{Project: a.Name(), Light: light, Dark: dark}, nil
	}
	light, dark := parseLegacyWebTheme(content)
	return ThemeObservation{Project: a.Name(), Light: light, Dark: dark}, nil
}

func (a WebAdapter) Plan(root string, cfg ThemeConfig) (ProjectPlan, error) {
	path := a.globalsPath(root)
	before, err := readFile(path)
	if err != nil {
		return ProjectPlan{}, err
	}
	beforeObs, err := a.Read(root)
	if err != nil {
		return ProjectPlan{}, err
	}

	themeBlock := buildWebThemeBlock(cfg)
	after := before
	if strings.Contains(after, webThemeStartMarker) && strings.Contains(after, webThemeEndMarker) {
		start := strings.Index(after, webThemeStartMarker)
		end := strings.Index(after, webThemeEndMarker)
		if start < 0 || end < 0 || end < start {
			return ProjectPlan{}, fmt.Errorf("managed web theme markers are malformed")
		}
		after = after[:start] + themeBlock + after[end+len(webThemeEndMarker):]
	} else {
		after = themeBlock + "\n\n" + after
	}
	after = normalizeWebGradients(after)

	plan := ProjectPlan{Project: a.Name()}
	addChange(&plan, path, before, after, "update web theme block")
	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "light.", beforeObs.Light, PaletteToMap(cfg.Colors.Light))...)
	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "dark.", beforeObs.Dark, PaletteToMap(cfg.Colors.Dark))...)
	return plan, nil
}

func (a WebAdapter) globalsPath(root string) string {
	return filepath.Join(root, webProjectName, "app", "globals.css")
}

func parseManagedWebTheme(content string) (map[string]string, map[string]string, error) {
	managed, err := extractManagedWebSection(content)
	if err != nil {
		return nil, nil, err
	}
	lightBlock, err := findSimpleCSSBlock(managed, ":root {", 0)
	if err != nil {
		return nil, nil, err
	}
	darkBlock, err := findSimpleCSSBlock(managed, `:root[data-uv-theme="dark"] {`, 0)
	if err != nil {
		return nil, nil, err
	}
	light, err := parseWebSemanticVariables(lightBlock.Body)
	if err != nil {
		return nil, nil, err
	}
	dark, err := parseWebSemanticVariables(darkBlock.Body)
	if err != nil {
		return nil, nil, err
	}
	return light, dark, nil
}

func extractManagedWebSection(content string) (string, error) {
	start := strings.Index(content, webThemeStartMarker)
	end := strings.Index(content, webThemeEndMarker)
	if start < 0 || end < 0 || end < start {
		return "", fmt.Errorf("managed web theme markers are missing")
	}
	return content[start : end+len(webThemeEndMarker)], nil
}

func parseWebSemanticVariables(block string) (map[string]string, error) {
	values := make(map[string]string)
	for _, role := range RoleOrder {
		name := "--uv-web-" + toKebab(role)
		pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `\s*:\s*(#[0-9A-Fa-f]{6})\s*;\s*$`)
		match := pattern.FindStringSubmatch(block)
		if len(match) != 2 {
			return nil, fmt.Errorf("managed web variable %s missing", name)
		}
		values[role] = strings.ToUpper(match[1])
	}
	return values, nil
}

func parseLegacyWebTheme(content string) (map[string]string, map[string]string) {
	light := parseLegacyWebMode(content, ":root {")
	dark := parseLegacyWebMode(content, `:root[data-uv-theme="dark"] {`)
	return light, dark
}

func parseLegacyWebMode(content, selector string) map[string]string {
	block, err := findSimpleCSSBlock(content, selector, 0)
	if err != nil {
		return map[string]string{}
	}
	legacyToRole := map[string]string{
		"--bg":      "background",
		"--panel":   "surface",
		"--text":    "text",
		"--muted":   "muted",
		"--primary": "primary",
		"--border":  "outline",
	}
	values := make(map[string]string)
	for variable, role := range legacyToRole {
		pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(variable) + `\s*:\s*(#[0-9A-Fa-f]{6})\s*;\s*$`)
		match := pattern.FindStringSubmatch(block.Body)
		if len(match) == 2 {
			values[role] = strings.ToUpper(match[1])
		}
	}
	return values
}

func buildWebThemeBlock(cfg ThemeConfig) string {
	light := PaletteToMap(cfg.Colors.Light)
	dark := PaletteToMap(cfg.Colors.Dark)
	builder := strings.Builder{}
	builder.WriteString(webThemeStartMarker)
	builder.WriteString("\n")
	builder.WriteString(serializeWebModeBlock(":root", light))
	builder.WriteString("\n\n")
	builder.WriteString(serializeWebModeBlock(`:root[data-uv-theme="dark"]`, dark))
	builder.WriteString("\n\n")
	builder.WriteString("@media (prefers-color-scheme: dark) {\n")
	inner := serializeWebModeBlock(`:root:not([data-uv-theme="light"])`, dark)
	for _, line := range strings.Split(inner, "\n") {
		if line == "" {
			continue
		}
		builder.WriteString("  " + line + "\n")
	}
	builder.WriteString("}\n")
	builder.WriteString(webThemeEndMarker)
	return builder.String()
}

func serializeWebModeBlock(selector string, palette map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString(selector + " {\n")
	for _, role := range RoleOrder {
		builder.WriteString("  --uv-web-" + toKebab(role) + ": " + strings.ToUpper(palette[role]) + ";\n")
	}
	builder.WriteString("\n")
	builder.WriteString("  --bg: var(--uv-web-background);\n")
	builder.WriteString("  --panel: var(--uv-web-surface);\n")
	builder.WriteString("  --text: var(--uv-web-text);\n")
	builder.WriteString("  --muted: var(--uv-web-muted);\n")
	builder.WriteString("  --primary: var(--uv-web-primary);\n")
	builder.WriteString("  --border: var(--uv-web-outline);\n")
	builder.WriteString("}\n")
	return strings.TrimRight(builder.String(), "\n")
}

func normalizeWebGradients(content string) string {
	content = strings.ReplaceAll(
		content,
		"radial-gradient(circle at top right, rgba(15, 118, 110, 0.1), transparent 45%),",
		"radial-gradient(circle at top right, color-mix(in srgb, var(--primary) 16%, transparent), transparent 45%),",
	)
	content = strings.ReplaceAll(
		content,
		"radial-gradient(circle at 15% 20%, rgba(56, 189, 248, 0.1), transparent 35%),",
		"radial-gradient(circle at 15% 20%, color-mix(in srgb, var(--muted) 20%, transparent), transparent 35%),",
	)
	return content
}
