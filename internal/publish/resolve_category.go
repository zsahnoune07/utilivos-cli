package publish

import (
	"fmt"
	"strings"
)

// ResolveCategoryNonInteractive picks category from manifest or flags when not using prompts.
// If the manifest category slug matches an API row, returns that row's slug and name.
// Otherwise --category-slug is required; if the slug matches a row, returns its name.
// If the slug is not in the catalog, --category-name is required (new category on publish).
func ResolveCategoryNonInteractive(m Manifest, cats []AdminCategory, slugFlag, nameFlag string) (slug, name string, err error) {
	want := Slugify(m.Category)
	for _, c := range cats {
		if Slugify(c.Slug) == want {
			return c.Slug, c.Name, nil
		}
	}
	sf := strings.TrimSpace(slugFlag)
	if sf == "" {
		return "", "", fmt.Errorf(
			"manifest category %q is not in the API catalog; set --category-slug (and --category-name when creating a new category), or run interactively without --non-interactive",
			want,
		)
	}
	normal := Slugify(sf)
	if !ValidCategorySlug(normal) {
		return "", "", fmt.Errorf("invalid --category-slug %q", sf)
	}
	for _, c := range cats {
		if Slugify(c.Slug) == normal {
			return c.Slug, c.Name, nil
		}
	}
	nm := strings.TrimSpace(nameFlag)
	if nm == "" {
		return "", "", fmt.Errorf("category %q is not in the API catalog; set --category-name for the display name of the new category", normal)
	}
	return normal, nm, nil
}
