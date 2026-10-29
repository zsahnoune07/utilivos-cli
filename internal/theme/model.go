package theme

import (
	"fmt"
	"sort"

	"github.com/utilivos/utilivos-cli/internal/update"
)

const SchemaVersion = 1

var RoleOrder = []string{
	"background",
	"surface",
	"primary",
	"onPrimary",
	"secondary",
	"onSecondary",
	"text",
	"muted",
	"error",
	"onError",
	"success",
	"warning",
	"surfaceAlt",
	"outline",
}

type ThemeConfig struct {
	SchemaVersion int         `json:"schemaVersion"`
	SeedColor     string      `json:"seedColor"`
	Colors        ThemeColors `json:"colors"`
}

type ThemeColors struct {
	Light ThemePalette `json:"light"`
	Dark  ThemePalette `json:"dark"`
}

type ThemePalette struct {
	Background  string `json:"background"`
	Surface     string `json:"surface"`
	Primary     string `json:"primary"`
	OnPrimary   string `json:"onPrimary"`
	Secondary   string `json:"secondary"`
	OnSecondary string `json:"onSecondary"`
	Text        string `json:"text"`
	Muted       string `json:"muted"`
	Error       string `json:"error"`
	OnError     string `json:"onError"`
	Success     string `json:"success"`
	Warning     string `json:"warning"`
	SurfaceAlt  string `json:"surfaceAlt"`
	Outline     string `json:"outline"`
}

func (p ThemePalette) Get(role string) (string, error) {
	switch role {
	case "background":
		return p.Background, nil
	case "surface":
		return p.Surface, nil
	case "primary":
		return p.Primary, nil
	case "onPrimary":
		return p.OnPrimary, nil
	case "secondary":
		return p.Secondary, nil
	case "onSecondary":
		return p.OnSecondary, nil
	case "text":
		return p.Text, nil
	case "muted":
		return p.Muted, nil
	case "error":
		return p.Error, nil
	case "onError":
		return p.OnError, nil
	case "success":
		return p.Success, nil
	case "warning":
		return p.Warning, nil
	case "surfaceAlt":
		return p.SurfaceAlt, nil
	case "outline":
		return p.Outline, nil
	default:
		return "", fmt.Errorf("unsupported role %q", role)
	}
}

func (p *ThemePalette) Set(role, value string) error {
	switch role {
	case "background":
		p.Background = value
	case "surface":
		p.Surface = value
	case "primary":
		p.Primary = value
	case "onPrimary":
		p.OnPrimary = value
	case "secondary":
		p.Secondary = value
	case "onSecondary":
		p.OnSecondary = value
	case "text":
		p.Text = value
	case "muted":
		p.Muted = value
	case "error":
		p.Error = value
	case "onError":
		p.OnError = value
	case "success":
		p.Success = value
	case "warning":
		p.Warning = value
	case "surfaceAlt":
		p.SurfaceAlt = value
	case "outline":
		p.Outline = value
	default:
		return fmt.Errorf("unsupported role %q", role)
	}
	return nil
}

type ThemeObservation struct {
	Project   string
	SeedColor string
	Light     map[string]string
	Dark      map[string]string
}

type ValueDiff struct {
	Project string
	Key     string
	Before  string
	After   string
}

type ProjectPlan struct {
	Project string
	Changes []update.FileChange
	Diffs   []ValueDiff
}

func SortDiffs(diffs []ValueDiff) {
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Project == diffs[j].Project {
			return diffs[i].Key < diffs[j].Key
		}
		return diffs[i].Project < diffs[j].Project
	})
}

type Detection struct {
	Project string
	Present bool
	Missing []string
}

type Adapter interface {
	Name() string
	Detect(root string) Detection
	Read(root string) (ThemeObservation, error)
	Plan(root string, cfg ThemeConfig) (ProjectPlan, error)
}
