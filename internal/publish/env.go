package publish

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/utilivos/utilivos-cli/internal/dotenv"
)

// LoadEnvFile loads dotenv from path if non-empty; otherwise tries cwd/.env when tryDefault is true.
func LoadEnvFile(cwd, explicitPath string, tryDefault bool) error {
	p := strings.TrimSpace(explicitPath)
	if p != "" {
		return dotenv.Load(p)
	}
	if !tryDefault {
		return nil
	}
	def := filepath.Join(cwd, ".env")
	if _, err := os.Stat(def); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return dotenv.Load(def)
}

// ResolveBaseURL returns UTILIVOS_API_BASE_URL or legacy UTILIVO_API_URL.
func ResolveBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("UTILIVOS_API_BASE_URL")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("UTILIVO_API_URL"))
}

// ResolveAPIKey returns UTILIVOS_ADMIN_API_KEY or legacy UTILIVO_ADMIN_API_KEY.
func ResolveAPIKey() string {
	if v := strings.TrimSpace(os.Getenv("UTILIVOS_ADMIN_API_KEY")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("UTILIVO_ADMIN_API_KEY"))
}
