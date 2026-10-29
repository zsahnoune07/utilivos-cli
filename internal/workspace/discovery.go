package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

var markers = []string{
	"utilivos-mobile",
	"utilivos-web",
	"utilivos-sdk",
	"docs/design-system/tokens.json",
}

type Workspace struct {
	Root string
}

func Discover(start string) (Workspace, error) {
	if start == "" {
		return Workspace{}, fmt.Errorf("workspace start path is required")
	}
	absStart, err := filepath.Abs(start)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve start path %s: %w", start, err)
	}
	resolvedStart, err := resolveDir(absStart)
	if err != nil {
		return Workspace{}, err
	}

	best := ""
	bestScore := 0
	for current := resolvedStart; ; current = filepath.Dir(current) {
		score := markerScore(current)
		if score > bestScore {
			best = current
			bestScore = score
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	if best == "" {
		return Workspace{}, fmt.Errorf("could not discover workspace root from %s", start)
	}
	return Workspace{Root: best}, nil
}

func resolveDir(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if st.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}

func markerScore(root string) int {
	score := 0
	for _, marker := range markers {
		if exists(filepath.Join(root, marker)) {
			score++
		}
	}
	return score
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
