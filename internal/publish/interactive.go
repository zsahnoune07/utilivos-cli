package publish

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// RunInteractivePrompts resolves category, statuses, and changelog using terminal prompts.
func RunInteractivePrompts(in terminal.FileReader, out terminal.FileWriter, errOut io.Writer, m Manifest, cats []AdminCategory) (
	categorySlug string,
	categoryName string,
	pluginStatus string,
	versionStatus string,
	changelog string,
	err error,
) {
	io := survey.WithStdio(in, out, errOut)
	want := Slugify(m.Category)
	var matched *AdminCategory
	for i := range cats {
		if Slugify(cats[i].Slug) == want {
			matched = &cats[i]
			break
		}
	}

	if matched != nil {
		if matched.Status == "archived" {
			_, _ = fmt.Fprintf(errOut, "\nNote: category %q (%s) is archived and hidden from the public catalog until reactivated.\n\n", matched.Name, matched.Slug)
		}
		use := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Use category %q (%s) from manifest?", matched.Name, matched.Slug),
			Default: true,
		}, &use, io); err != nil {
			return "", "", "", "", "", err
		}
		if use {
			categorySlug, categoryName = matched.Slug, matched.Name
		}
	}

	if categorySlug == "" {
		choices := make([]string, 0, len(cats)+1)
		sort.Slice(cats, func(i, j int) bool { return cats[i].Name < cats[j].Name })
		for _, c := range cats {
			label := fmt.Sprintf("%s (%s)", c.Name, c.Slug)
			if c.Status == "archived" {
				label += " — archived"
			}
			choices = append(choices, label)
		}
		choices = append(choices, fmt.Sprintf("Create new category (manifest slug: %s)", want))

		var picked string
		prompt := &survey.Select{
			Message: "Choose a category",
			Options: choices,
		}
		if err := survey.AskOne(prompt, &picked, io); err != nil {
			return "", "", "", "", "", err
		}
		if picked == choices[len(choices)-1] {
			defSlug := want
			if defSlug == "" {
				defSlug = "utilities"
			}
			if err := survey.AskOne(&survey.Input{Message: "New category slug (kebab-case)", Default: defSlug}, &categorySlug, io); err != nil {
				return "", "", "", "", "", err
			}
			categorySlug = Slugify(categorySlug)
			if !ValidCategorySlug(categorySlug) {
				return "", "", "", "", "", fmt.Errorf("invalid category slug")
			}
			defName := humanizeSlug(categorySlug)
			if err := survey.AskOne(&survey.Input{Message: "New category display name", Default: defName}, &categoryName, io); err != nil {
				return "", "", "", "", "", err
			}
			categoryName = strings.TrimSpace(categoryName)
			if categoryName == "" {
				categoryName = defName
			}
		} else {
			idx := indexOfString(choices, picked)
			if idx < 0 || idx >= len(choices)-1 {
				return "", "", "", "", "", fmt.Errorf("invalid selection")
			}
			categorySlug = cats[idx].Slug
			categoryName = cats[idx].Name
		}
	}

	if err := survey.AskOne(&survey.Select{
		Message: "Plugin catalog status",
		Options: []string{"draft", "published", "archived"},
		Default: "draft",
	}, &pluginStatus, io); err != nil {
		return "", "", "", "", "", err
	}
	if err := survey.AskOne(&survey.Select{
		Message: "Version status (this upload)",
		Options: []string{"draft", "published", "deprecated"},
		Default: "draft",
	}, &versionStatus, io); err != nil {
		return "", "", "", "", "", err
	}
	if err := survey.AskOne(&survey.Input{Message: "Changelog (optional, empty to skip)", Default: ""}, &changelog, io); err != nil {
		return "", "", "", "", "", err
	}
	changelog = strings.TrimSpace(changelog)

	return categorySlug, categoryName, pluginStatus, versionStatus, changelog, nil
}

func indexOfString(slice []string, v string) int {
	for i, s := range slice {
		if s == v {
			return i
		}
	}
	return -1
}
