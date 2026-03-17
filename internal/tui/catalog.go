package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"gopkg.in/yaml.v3"
)

type TemplateEntry struct {
	ID                 string
	Category           string
	SupportedSizes     []domain.PageSize
	DefaultOrientation domain.Orientation
	Source             string
}

func loadTemplateEntries() []TemplateEntry {
	root := templateDir()
	if root == "" {
		return nil
	}

	entries := []TemplateEntry{}

	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		normalized := filepath.ToSlash(relPath)

		if d.IsDir() {
			if normalized == "." {
				return nil
			}
			if strings.HasPrefix(normalized, "assets") || strings.HasPrefix(normalized, "samples") {
				return fs.SkipDir
			}
			return nil
		}

		if filepath.Ext(d.Name()) != ".yaml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var tmpl domain.Template
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return nil
		}

		if tmpl.ID == "" {
			return nil
		}

		category := deriveCategory(normalized)

		entry := TemplateEntry{
			ID:                 tmpl.ID,
			Category:           category,
			SupportedSizes:     tmpl.Page.SupportedSizes,
			DefaultOrientation: tmpl.Page.DefaultOrientation,
			Source:             path,
		}

		if entry.DefaultOrientation == "" {
			entry.DefaultOrientation = domain.OrientationPortrait
		}

		if len(entry.SupportedSizes) == 0 {
			entry.SupportedSizes = []domain.PageSize{domain.PageSizeA4}
		}

		entries = append(entries, entry)
		return nil
	}

	_ = filepath.WalkDir(root, walk)

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category == entries[j].Category {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Category < entries[j].Category
	})

	return entries
}

func deriveCategory(relPath string) string {
	if relPath == "" {
		return ""
	}

	segments := strings.Split(relPath, "/")
	if len(segments) == 0 {
		return ""
	}

	return segments[0]
}

func (e TemplateEntry) Label() string {
	if e.Category != "" {
		return e.ID + " (" + strings.Title(e.Category) + ")"
	}
	return e.ID
}

func templateDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return filepath.Join(projectRoot, "templates")
}
