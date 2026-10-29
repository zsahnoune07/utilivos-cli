package theme

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type ValidationError struct {
	Issues []string
}

func (e ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "invalid theme config"
	}
	return "invalid theme config:\n - " + strings.Join(e.Issues, "\n - ")
}

func LoadConfig(path string) (ThemeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := DecodeConfig(data)
	if err != nil {
		return ThemeConfig{}, err
	}
	return cfg, nil
}

func DecodeConfig(data []byte) (ThemeConfig, error) {
	if err := detectDuplicateKeys(data); err != nil {
		return ThemeConfig{}, err
	}
	var cfg ThemeConfig
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return ThemeConfig{}, fmt.Errorf("parse config JSON: %w", err)
	}
	if err := ensureNoTrailingJSON(dec); err != nil {
		return ThemeConfig{}, err
	}
	if err := ValidateAndNormalize(&cfg); err != nil {
		return ThemeConfig{}, err
	}
	return cfg, nil
}

func ensureNoTrailingJSON(dec *json.Decoder) error {
	var trailing any
	err := dec.Decode(&trailing)
	if err == nil {
		return fmt.Errorf("parse config JSON: trailing content after root object")
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return fmt.Errorf("parse config JSON: %w", err)
}

func ValidateAndNormalize(cfg *ThemeConfig) error {
	issues := make([]string, 0)
	if cfg.SchemaVersion != SchemaVersion {
		issues = append(issues, fmt.Sprintf("$.schemaVersion must be %d", SchemaVersion))
	}

	seed, err := normalizeHexColor(cfg.SeedColor)
	if err != nil {
		issues = append(issues, fmt.Sprintf("$.seedColor %s", err.Error()))
	} else {
		cfg.SeedColor = seed
	}

	validatePalette := func(mode string, palette *ThemePalette) {
		for _, role := range RoleOrder {
			value, getErr := palette.Get(role)
			if getErr != nil {
				issues = append(issues, fmt.Sprintf("$.colors.%s.%s %s", mode, role, getErr.Error()))
				continue
			}
			normalized, colorErr := normalizeHexColor(value)
			if colorErr != nil {
				issues = append(issues, fmt.Sprintf("$.colors.%s.%s %s", mode, role, colorErr.Error()))
				continue
			}
			if setErr := palette.Set(role, normalized); setErr != nil {
				issues = append(issues, fmt.Sprintf("$.colors.%s.%s %s", mode, role, setErr.Error()))
			}
		}
	}

	validatePalette("light", &cfg.Colors.Light)
	validatePalette("dark", &cfg.Colors.Dark)

	if len(issues) > 0 {
		return ValidationError{Issues: issues}
	}
	return nil
}

func normalizeHexColor(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", fmt.Errorf("is required")
	}
	if !hexColorPattern.MatchString(value) {
		return "", fmt.Errorf("must match #RRGGBB")
	}
	return strings.ToUpper(value), nil
}

func EncodeConfig(cfg ThemeConfig) ([]byte, error) {
	copyCfg := cfg
	if err := ValidateAndNormalize(&copyCfg); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(copyCfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode config JSON: %w", err)
	}
	return append(out, '\n'), nil
}

func ResolveConfigPath(workspaceRoot, configPath string) string {
	if filepath.IsAbs(configPath) {
		return configPath
	}
	return filepath.Join(workspaceRoot, configPath)
}
