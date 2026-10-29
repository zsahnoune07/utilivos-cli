package publish

import (
	"regexp"
	"strings"
)

var kebabSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slugify matches utilivos-admin lib/slug.ts: lowercase kebab-case for API slugs.
func Slugify(value string) string {
	s := strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

// ValidCategorySlug reports whether s is non-empty kebab-case (API category slug pattern).
func ValidCategorySlug(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && kebabSlugPattern.MatchString(s)
}
