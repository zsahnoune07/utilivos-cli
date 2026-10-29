package theme

import (
	"fmt"
	"strings"
)

func PaletteToMap(p ThemePalette) map[string]string {
	out := make(map[string]string, len(RoleOrder))
	for _, role := range RoleOrder {
		value, err := p.Get(role)
		if err != nil {
			continue
		}
		out[role] = strings.ToUpper(value)
	}
	return out
}

func MapToPalette(values map[string]string) (ThemePalette, error) {
	var palette ThemePalette
	for _, role := range RoleOrder {
		value, ok := values[role]
		if !ok {
			return ThemePalette{}, fmt.Errorf("missing role %q", role)
		}
		if err := palette.Set(role, value); err != nil {
			return ThemePalette{}, err
		}
	}
	return palette, nil
}

func NewConfigFromCanonical(seed string, light, dark map[string]string) (ThemeConfig, error) {
	lightPalette, err := MapToPalette(light)
	if err != nil {
		return ThemeConfig{}, fmt.Errorf("build light palette: %w", err)
	}
	darkPalette, err := MapToPalette(dark)
	if err != nil {
		return ThemeConfig{}, fmt.Errorf("build dark palette: %w", err)
	}
	cfg := ThemeConfig{
		SchemaVersion: SchemaVersion,
		SeedColor:     seed,
		Colors: ThemeColors{
			Light: lightPalette,
			Dark:  darkPalette,
		},
	}
	if err := ValidateAndNormalize(&cfg); err != nil {
		return ThemeConfig{}, err
	}
	return cfg, nil
}
