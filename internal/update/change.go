package update

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type FileChange struct {
	Path        string
	Before      string
	After       string
	Description string
}

func SortChanges(changes []FileChange) {
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})
}

func Apply(changes []FileChange) error {
	SortChanges(changes)
	for _, change := range changes {
		if change.Before == change.After {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(change.Path), 0o755); err != nil {
			return fmt.Errorf("create parent for %s: %w", change.Path, err)
		}
		if err := os.WriteFile(change.Path, []byte(change.After), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", change.Path, err)
		}
	}
	return nil
}
