package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	DefaultThemeConfigPath string `json:"defaultThemeConfigPath"`
}

func DefaultSettings() Settings {
	return Settings{DefaultThemeConfigPath: "utilivos.theme.json"}
}

func LoadSettings(workspaceRoot string) (Settings, error) {
	settings := DefaultSettings()
	path := filepath.Join(workspaceRoot, ".utilivos-cli.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return settings, fmt.Errorf("read settings file %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&settings); err != nil {
		return settings, fmt.Errorf("parse settings file %s: %w", path, err)
	}
	if settings.DefaultThemeConfigPath == "" {
		settings.DefaultThemeConfigPath = "utilivos.theme.json"
	}
	return settings, nil
}
