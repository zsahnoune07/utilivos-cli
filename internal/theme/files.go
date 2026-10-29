package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/utilivos/utilivos-cli/internal/update"
)

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func addChange(plan *ProjectPlan, path, before, after, description string) {
	if before == after {
		return
	}
	plan.Changes = append(plan.Changes, update.FileChange{
		Path:        path,
		Before:      before,
		After:       after,
		Description: description,
	})
}

func roleDiffs(project, prefix string, before, after map[string]string) []ValueDiff {
	keys := make([]string, 0, len(before)+len(after))
	seen := make(map[string]struct{})
	for key := range before {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range after {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	diffs := make([]ValueDiff, 0)
	for _, key := range keys {
		b := strings.ToUpper(strings.TrimSpace(before[key]))
		a := strings.ToUpper(strings.TrimSpace(after[key]))
		if b == a {
			continue
		}
		diffs = append(diffs, ValueDiff{
			Project: project,
			Key:     prefix + key,
			Before:  b,
			After:   a,
		})
	}
	return diffs
}

func absolute(root string, elements ...string) string {
	parts := append([]string{root}, elements...)
	return filepath.Join(parts...)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
