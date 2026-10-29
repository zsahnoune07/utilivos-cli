package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDoesNotOverrideExisting(t *testing.T) {
	t.Setenv("UTILIVOS_TEST_DOTENV", "from-env")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	other := "UTILIVOS_TEST_DOTENV_OTHER_" + t.Name()
	content := "UTILIVOS_TEST_DOTENV=from-file\n" + other + "=set\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UTILIVOS_TEST_DOTENV") != "from-env" {
		t.Fatalf("existing env overridden: %q", os.Getenv("UTILIVOS_TEST_DOTENV"))
	}
	if os.Getenv(other) != "set" {
		t.Fatalf("%s: %q", other, os.Getenv(other))
	}
}

func TestLoadSetsWhenUnset(t *testing.T) {
	key := "UTILIVOS_TEST_DOTENV_UNSET_" + t.Name()
	_ = os.Unsetenv(key)
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(key+"=hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(key) != "hello" {
		t.Fatalf("got %q", os.Getenv(key))
	}
}
