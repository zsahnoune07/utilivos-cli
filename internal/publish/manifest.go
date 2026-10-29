package publish

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxBundleBytes = 256 << 20 // 256 MiB, aligned with utilivos-api

var (
	slugPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
)

// Manifest holds fields needed to publish; JSON matches plugin manifest.json.
type Manifest struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Version         string `json:"version"`
	Description     string `json:"description"`
	LongDescription string `json:"long_description"`
	Category        string `json:"category"`
	Tier            string `json:"tier"`
}

var manifestZipNames = []string{"manifest.json", "manifest"}

// ReadManifestFromUBB opens a .ubb zip and parses the first present manifest file (root paths only).
func ReadManifestFromUBB(ubbPath string) (Manifest, error) {
	var zero Manifest
	if strings.TrimSpace(ubbPath) == "" {
		return zero, fmt.Errorf("bundle path is empty")
	}
	ext := strings.ToLower(filepath.Ext(ubbPath))
	if ext != ".ubb" {
		return zero, fmt.Errorf("expected .ubb file, got %q", ext)
	}
	zr, err := zip.OpenReader(ubbPath)
	if err != nil {
		return zero, fmt.Errorf("open bundle: %w", err)
	}
	defer zr.Close()

	for _, wantName := range manifestZipNames {
		for _, f := range zr.File {
			if f.Name != wantName {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return zero, fmt.Errorf("read %s: %w", wantName, err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return zero, fmt.Errorf("read %s: %w", wantName, err)
			}
			var m Manifest
			if err := json.Unmarshal(b, &m); err != nil {
				return zero, fmt.Errorf("parse %s: %w", wantName, err)
			}
			return m, nil
		}
	}
	return zero, fmt.Errorf("manifest not found (expected manifest.json or manifest at archive root)")
}

// ValidateForPublish checks fields required by the admin publish API.
func ValidateForPublish(m Manifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("manifest name is required")
	}
	slug := Slugify(m.Slug)
	if slug == "" {
		return fmt.Errorf("manifest slug is required")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("manifest slug must be lowercase kebab-case")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	if !semverPattern.MatchString(strings.TrimSpace(m.Version)) {
		return fmt.Errorf("manifest version must be semver (e.g. 1.0.0)")
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("manifest description is required")
	}
	cat := Slugify(m.Category)
	if cat == "" {
		return fmt.Errorf("manifest category is required")
	}
	if !ValidCategorySlug(cat) {
		return fmt.Errorf("manifest category must be kebab-case (letters, digits, hyphens)")
	}
	tier := strings.ToLower(strings.TrimSpace(m.Tier))
	if tier != "" && tier != "free" && tier != "premium" {
		return fmt.Errorf("manifest tier must be free, premium, or empty (defaults to free)")
	}
	return nil
}

// CheckBundleSize fails if the file is missing or exceeds API limit.
func CheckBundleSize(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat bundle: %w", err)
	}
	if st.Size() > maxBundleBytes {
		return fmt.Errorf("bundle exceeds %d MiB API limit", maxBundleBytes>>20)
	}
	if st.Size() == 0 {
		return fmt.Errorf("bundle is empty")
	}
	return nil
}
