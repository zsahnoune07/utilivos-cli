package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFromRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "utilivos-mobile"))
	mustMkdirAll(t, filepath.Join(root, "utilivos-web"))
	mustMkdirAll(t, filepath.Join(root, "utilivos-sdk"))
	mustWriteFile(t, filepath.Join(root, "docs", "design-system", "tokens.json"), "{}")

	ws, err := Discover(root)
	if err != nil {
		t.Fatalf("discover workspace: %v", err)
	}
	if ws.Root != root {
		t.Fatalf("expected root %s, got %s", root, ws.Root)
	}
}

func TestDiscoverFromNestedPath(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "utilivos-mobile", "lib", "core"))
	mustMkdirAll(t, filepath.Join(root, "utilivos-web"))
	mustMkdirAll(t, filepath.Join(root, "utilivos-sdk"))
	mustWriteFile(t, filepath.Join(root, "docs", "design-system", "tokens.json"), "{}")

	nested := filepath.Join(root, "utilivos-mobile", "lib", "core")
	ws, err := Discover(nested)
	if err != nil {
		t.Fatalf("discover workspace: %v", err)
	}
	if ws.Root != root {
		t.Fatalf("expected root %s, got %s", root, ws.Root)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
