package theme

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const sdkProjectName = "utilivos-sdk"

type sdkTokenDocument struct {
	Source struct {
		SeedColor string `json:"seed_color"`
	} `json:"source"`
	Colors struct {
		Light ThemePalette `json:"light"`
		Dark  ThemePalette `json:"dark"`
	} `json:"colors"`
}

type SDKAdapter struct{}

func (a SDKAdapter) Name() string {
	return sdkProjectName
}

func (a SDKAdapter) Detect(root string) Detection {
	required := []string{
		filepath.Join(root, "docs", "design-system", "tokens.json"),
		filepath.Join(root, sdkProjectName, "ui", "base.css"),
		filepath.Join(root, sdkProjectName, "internal", "templates", "assets", "base.css"),
	}
	missing := make([]string, 0)
	for _, path := range required {
		if !exists(path) {
			missing = append(missing, path)
		}
	}
	return Detection{Project: a.Name(), Present: len(missing) == 0, Missing: missing}
}

func (a SDKAdapter) Read(root string) (ThemeObservation, error) {
	tokensPath := a.tokensPath(root)
	content, err := readFile(tokensPath)
	if err != nil {
		return ThemeObservation{}, err
	}
	var doc sdkTokenDocument
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return ThemeObservation{}, fmt.Errorf("parse %s: %w", tokensPath, err)
	}
	observation := ThemeObservation{
		Project:   a.Name(),
		SeedColor: strings.ToUpper(strings.TrimSpace(doc.Source.SeedColor)),
		Light:     PaletteToMap(doc.Colors.Light),
		Dark:      PaletteToMap(doc.Colors.Dark),
	}
	if observation.SeedColor != "" && !strings.HasPrefix(observation.SeedColor, "#") {
		observation.SeedColor = "#" + observation.SeedColor
	}
	return observation, nil
}

func (a SDKAdapter) Plan(root string, cfg ThemeConfig) (ProjectPlan, error) {
	tokensPath := a.tokensPath(root)
	tokensBefore, err := readFile(tokensPath)
	if err != nil {
		return ProjectPlan{}, err
	}
	beforeObs, err := a.Read(root)
	if err != nil {
		return ProjectPlan{}, err
	}

	tokensAfter, err := replaceSDKTokenFile(tokensBefore, cfg)
	if err != nil {
		return ProjectPlan{}, err
	}

	plan := ProjectPlan{Project: a.Name()}
	addChange(&plan, tokensPath, tokensBefore, tokensAfter, "update SDK token source")

	cssTargets, err := a.cssPaths(root)
	if err != nil {
		return ProjectPlan{}, err
	}
	for _, cssPath := range cssTargets {
		before, readErr := readFile(cssPath)
		if readErr != nil {
			return ProjectPlan{}, readErr
		}
		after, updateErr := updateSDKBaseCSS(before, cfg)
		if updateErr != nil {
			return ProjectPlan{}, fmt.Errorf("update %s: %w", cssPath, updateErr)
		}
		addChange(&plan, cssPath, before, after, "refresh SDK base.css color variables")
	}

	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "light.", beforeObs.Light, PaletteToMap(cfg.Colors.Light))...)
	plan.Diffs = append(plan.Diffs, roleDiffs(a.Name(), "dark.", beforeObs.Dark, PaletteToMap(cfg.Colors.Dark))...)
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

func (a SDKAdapter) tokensPath(root string) string {
	return filepath.Join(root, "docs", "design-system", "tokens.json")
}

func (a SDKAdapter) cssPaths(root string) ([]string, error) {
	paths := []string{
		filepath.Join(root, sdkProjectName, "ui", "base.css"),
		filepath.Join(root, sdkProjectName, "internal", "templates", "assets", "base.css"),
	}
	globs := []string{
		filepath.Join(root, sdkProjectName, "examples", "*", "tool", "utilivos", "base.css"),
		filepath.Join(root, sdkProjectName, "tests", "fixtures", "*", "tool", "utilivos", "base.css"),
	}
	for _, pattern := range globs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		paths = append(paths, matches...)
	}
	unique := make(map[string]struct{})
	finalPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := unique[path]; ok {
			continue
		}
		unique[path] = struct{}{}
		finalPaths = append(finalPaths, path)
	}
	sort.Strings(finalPaths)
	return finalPaths, nil
}

func replaceSDKTokenFile(content string, cfg ThemeConfig) (string, error) {
	updated, err := replaceJSONHexValue(content, "seed_color", cfg.SeedColor)
	if err != nil {
		return "", err
	}
	updated, err = replaceColorsObject(updated, "light", PaletteToMap(cfg.Colors.Light))
	if err != nil {
		return "", err
	}
	updated, err = replaceColorsObject(updated, "dark", PaletteToMap(cfg.Colors.Dark))
	if err != nil {
		return "", err
	}
	return updated, nil
}

func replaceJSONHexValue(content, key, value string) (string, error) {
	pattern := regexp.MustCompile(`(?m)("` + regexp.QuoteMeta(key) + `"\s*:\s*")#?[0-9A-Fa-f]{6}(")`)
	if !pattern.MatchString(content) {
		return "", fmt.Errorf("JSON key %q not found", key)
	}
	normalized := strings.TrimPrefix(strings.ToUpper(value), "#")
	return pattern.ReplaceAllString(content, "${1}#"+normalized+"${2}"), nil
}

func replaceColorsObject(content, objectName string, palette map[string]string) (string, error) {
	_, colorsOpen, colorsEnd, err := findNamedJSONObject(content, "colors", 0)
	if err != nil {
		return "", err
	}
	colorsBlock := content[colorsOpen:colorsEnd]
	_, objectOpen, objectEnd, err := findNamedJSONObject(colorsBlock, objectName, 0)
	if err != nil {
		return "", err
	}
	objectBlock := colorsBlock[objectOpen:objectEnd]
	updatedBlock := objectBlock
	for _, role := range RoleOrder {
		pattern := regexp.MustCompile(`(?m)("` + regexp.QuoteMeta(role) + `"\s*:\s*")#?[0-9A-Fa-f]{6}(")`)
		if !pattern.MatchString(updatedBlock) {
			return "", fmt.Errorf("role %q missing in %s object", role, objectName)
		}
		normalized := strings.TrimPrefix(strings.ToUpper(palette[role]), "#")
		updatedBlock = pattern.ReplaceAllString(updatedBlock, "${1}#"+normalized+"${2}")
	}
	if updatedBlock == objectBlock {
		return content, nil
	}
	updatedColorsBlock := colorsBlock[:objectOpen] + updatedBlock + colorsBlock[objectEnd:]
	return content[:colorsOpen] + updatedColorsBlock + content[colorsEnd:], nil
}

func findNamedJSONObject(content, name string, startAt int) (nameIndex, objectStart, objectEnd int, err error) {
	token := `"` + name + `"`
	search := content[startAt:]
	rel := strings.Index(search, token)
	if rel < 0 {
		return 0, 0, 0, fmt.Errorf("JSON object %q not found", name)
	}
	nameIndex = startAt + rel
	colon := strings.Index(content[nameIndex+len(token):], ":")
	if colon < 0 {
		return 0, 0, 0, fmt.Errorf("JSON key %q missing colon", name)
	}
	colonIndex := nameIndex + len(token) + colon
	openRel := strings.Index(content[colonIndex:], "{")
	if openRel < 0 {
		return 0, 0, 0, fmt.Errorf("JSON object %q missing opening brace", name)
	}
	objectStart = colonIndex + openRel
	objectEnd, err = findMatchingBrace(content, objectStart)
	if err != nil {
		return 0, 0, 0, err
	}
	return nameIndex, objectStart, objectEnd + 1, nil
}

func findMatchingBrace(content string, openIndex int) (int, error) {
	depth := 0
	inString := false
	escaped := false
	for index := openIndex; index < len(content); index++ {
		char := content[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
			continue
		}
		if char == '{' {
			depth++
			continue
		}
		if char == '}' {
			depth--
			if depth == 0 {
				return index, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced JSON braces")
}

func updateSDKBaseCSS(content string, cfg ThemeConfig) (string, error) {
	updated := content
	lightMap := PaletteToMap(cfg.Colors.Light)
	darkMap := PaletteToMap(cfg.Colors.Dark)

	block, err := findRootColorBlock(updated)
	if err != nil {
		return "", err
	}
	updatedLight, err := replaceCSSColorVariables(block.Body, lightMap)
	if err != nil {
		return "", fmt.Errorf("replace light color block: %w", err)
	}
	updated = updated[:block.Start] + updatedLight + updated[block.End:]

	darkBlock, err := findSimpleCSSBlock(updated, `:root[data-uv-theme="dark"] {`, 0)
	if err != nil {
		return "", err
	}
	updatedDark, err := replaceCSSColorVariables(darkBlock.Body, darkMap)
	if err != nil {
		return "", fmt.Errorf("replace dark color block: %w", err)
	}
	updated = updated[:darkBlock.Start] + updatedDark + updated[darkBlock.End:]

	mediaBlock, err := findSimpleCSSBlock(updated, `:root:not([data-uv-theme="light"]) {`, darkBlock.Start)
	if err != nil {
		return "", err
	}
	updatedMedia, err := replaceCSSColorVariables(mediaBlock.Body, darkMap)
	if err != nil {
		return "", fmt.Errorf("replace media dark color block: %w", err)
	}
	updated = updated[:mediaBlock.Start] + updatedMedia + updated[mediaBlock.End:]

	return updated, nil
}

type cssBlock struct {
	Start int
	End   int
	Body  string
}

func findRootColorBlock(content string) (cssBlock, error) {
	searchAt := 0
	for {
		block, err := findSimpleCSSBlock(content, ":root {", searchAt)
		if err != nil {
			return cssBlock{}, err
		}
		if strings.Contains(block.Body, "--uv-color-background") {
			return block, nil
		}
		searchAt = block.End
		if searchAt >= len(content) {
			return cssBlock{}, fmt.Errorf("light :root color block not found")
		}
	}
}

func findSimpleCSSBlock(content, selector string, startAt int) (cssBlock, error) {
	rel := strings.Index(content[startAt:], selector)
	if rel < 0 {
		return cssBlock{}, fmt.Errorf("CSS selector %q not found", selector)
	}
	start := startAt + rel
	open := start + len(selector) - 1
	if open < 0 || open >= len(content) || content[open] != '{' {
		return cssBlock{}, fmt.Errorf("invalid CSS block for selector %q", selector)
	}
	close := strings.Index(content[open:], "}")
	if close < 0 {
		return cssBlock{}, fmt.Errorf("CSS block for selector %q is not terminated", selector)
	}
	end := open + close + 1
	return cssBlock{Start: start, End: end, Body: content[start:end]}, nil
}

func replaceCSSColorVariables(block string, palette map[string]string) (string, error) {
	updated := block
	for _, role := range RoleOrder {
		name := "--uv-color-" + toKebab(role)
		pattern := regexp.MustCompile(`(?m)(^\s*` + regexp.QuoteMeta(name) + `\s*:\s*)#[0-9A-Fa-f]{6}(\s*;\s*$)`)
		if !pattern.MatchString(updated) {
			return "", fmt.Errorf("CSS variable %s missing", name)
		}
		hex := strings.TrimPrefix(strings.ToLower(palette[role]), "#")
		updated = pattern.ReplaceAllString(updated, "${1}#"+hex+"${2}")
	}
	return updated, nil
}

func toKebab(value string) string {
	builder := strings.Builder{}
	for index, char := range value {
		if index > 0 && char >= 'A' && char <= 'Z' {
			builder.WriteByte('-')
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}
