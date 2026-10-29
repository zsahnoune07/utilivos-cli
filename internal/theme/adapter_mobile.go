package theme

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const mobileProjectName = "utilivos-mobile"

var mobileRolePattern = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)\s*:\s*Color\(0xFF([0-9A-Fa-f]{6})\),\s*$`)
var mobileSeedPattern = regexp.MustCompile(`(?m)^\s*static\s+const\s+Color\s+seedColor\s*=\s*Color\(0xFF([0-9A-Fa-f]{6})\);\s*$`)

type MobileAdapter struct{}

func (a MobileAdapter) Name() string {
	return mobileProjectName
}

func (a MobileAdapter) Detect(root string) Detection {
	path := a.tokenPath(root)
	missing := make([]string, 0)
	if !exists(path) {
		missing = append(missing, path)
	}
	return Detection{Project: a.Name(), Present: len(missing) == 0, Missing: missing}
}

func (a MobileAdapter) Read(root string) (ThemeObservation, error) {
	path := a.tokenPath(root)
	content, err := readFile(path)
	if err != nil {
		return ThemeObservation{}, err
	}
	seed, err := parseMobileSeed(content)
	if err != nil {
		return ThemeObservation{}, fmt.Errorf("parse mobile seed: %w", err)
	}
	light, err := parseMobilePalette(content, "light")
	if err != nil {
		return ThemeObservation{}, fmt.Errorf("parse mobile light palette: %w", err)
	}
	dark, err := parseMobilePalette(content, "dark")
	if err != nil {
		return ThemeObservation{}, fmt.Errorf("parse mobile dark palette: %w", err)
	}
	return ThemeObservation{Project: a.Name(), SeedColor: seed, Light: light, Dark: dark}, nil
}

func (a MobileAdapter) Plan(root string, cfg ThemeConfig) (ProjectPlan, error) {
	path := a.tokenPath(root)
	before, err := readFile(path)
	if err != nil {
		return ProjectPlan{}, err
	}
	beforeObs, err := a.Read(root)
	if err != nil {
		return ProjectPlan{}, err
	}

	after := before
	after, err = replaceMobileSeed(after, cfg.SeedColor)
	if err != nil {
		return ProjectPlan{}, err
	}
	after, err = replaceMobilePalette(after, "light", PaletteToMap(cfg.Colors.Light))
	if err != nil {
		return ProjectPlan{}, err
	}
	after, err = replaceMobilePalette(after, "dark", PaletteToMap(cfg.Colors.Dark))
	if err != nil {
		return ProjectPlan{}, err
	}

	plan := ProjectPlan{Project: a.Name()}
	addChange(&plan, path, before, after, "update mobile theme tokens")

	afterLight := PaletteToMap(cfg.Colors.Light)
	afterDark := PaletteToMap(cfg.Colors.Dark)
	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "light.", beforeObs.Light, afterLight)...)
	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "dark.", beforeObs.Dark, afterDark)...)
	if strings.ToUpper(beforeObs.SeedColor) != strings.ToUpper(cfg.SeedColor) {
		plan.Diffs = append(plan.Diffs, ValueDiff{
			Project: a.Name(),
			Key:     "seedColor",
			Before:  strings.ToUpper(beforeObs.SeedColor),
			After:   strings.ToUpper(cfg.SeedColor),
		})
	}
	return plan, nil
}

func (a MobileAdapter) tokenPath(root string) string {
	return filepath.Join(root, mobileProjectName, "lib", "core", "theme", "app_design_tokens.dart")
}

func parseMobileSeed(content string) (string, error) {
	match := mobileSeedPattern.FindStringSubmatch(content)
	if len(match) != 2 {
		return "", fmt.Errorf("seedColor declaration not found")
	}
	return "#" + strings.ToUpper(match[1]), nil
}

func parseMobilePalette(content, mode string) (map[string]string, error) {
	block, err := extractMobilePaletteBlock(content, mode)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	matches := mobileRolePattern.FindAllStringSubmatch(block, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		key := match[1]
		values[key] = "#" + strings.ToUpper(match[2])
	}
	for _, role := range RoleOrder {
		if _, ok := values[role]; !ok {
			return nil, fmt.Errorf("role %q missing in %s palette", role, mode)
		}
	}
	return values, nil
}

func extractMobilePaletteBlock(content, mode string) (string, error) {
	startToken := "static const AppColorPalette " + mode + " = AppColorPalette("
	start := strings.Index(content, startToken)
	if start < 0 {
		return "", fmt.Errorf("%s palette declaration not found", mode)
	}
	rest := content[start+len(startToken):]
	endRelative := strings.Index(rest, ");")
	if endRelative < 0 {
		return "", fmt.Errorf("%s palette terminator not found", mode)
	}
	return rest[:endRelative], nil
}

func replaceMobileSeed(content, seedColor string) (string, error) {
	hex := strings.TrimPrefix(strings.ToUpper(seedColor), "#")
	if !mobileSeedPattern.MatchString(content) {
		return "", fmt.Errorf("seedColor declaration not found for replacement")
	}
	updated := mobileSeedPattern.ReplaceAllString(content, "  static const Color seedColor = Color(0xFF"+hex+");")
	return updated, nil
}

func replaceMobilePalette(content, mode string, palette map[string]string) (string, error) {
	startToken := "static const AppColorPalette " + mode + " = AppColorPalette("
	start := strings.Index(content, startToken)
	if start < 0 {
		return "", fmt.Errorf("%s palette declaration not found for replacement", mode)
	}
	contentAfterStart := content[start+len(startToken):]
	endRelative := strings.Index(contentAfterStart, ");")
	if endRelative < 0 {
		return "", fmt.Errorf("%s palette terminator not found for replacement", mode)
	}
	block := contentAfterStart[:endRelative]

	updatedBlock := block
	for _, role := range RoleOrder {
		value := strings.TrimPrefix(strings.ToUpper(palette[role]), "#")
		pattern := regexp.MustCompile(`(?m)(^\s*` + regexp.QuoteMeta(role) + `\s*:\s*Color\(0xFF)([0-9A-Fa-f]{6})(\),\s*$)`)
		if !pattern.MatchString(updatedBlock) {
			return "", fmt.Errorf("role %q missing in %s palette replacement", role, mode)
		}
		updatedBlock = pattern.ReplaceAllString(updatedBlock, "${1}"+value+"${3}")
	}

	if updatedBlock == block {
		return content, nil
	}
	return content[:start+len(startToken)] + updatedBlock + contentAfterStart[endRelative:], nil
}
