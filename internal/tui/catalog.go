package tui

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/schema"
)

var primaryImageSlotPriority = []string{
	"artwork",
	"photo",
	"hero",
	"image",
	"cover",
}

type TemplateEntry struct {
	ID                 string
	Category           string
	SupportedSizes     []domain.PageSize
	DefaultOrientation domain.Orientation
	Source             string
	ImageSlots         []domain.Slot
	DecorationAssets   []domain.Asset
	Template           domain.Template
}

func loadTemplateEntries() []TemplateEntry {
	root := templateDir()
	if root == "" {
		return nil
	}

	entries := make([]TemplateEntry, 0)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			relPath, relErr := filepath.Rel(root, path)
			if relErr != nil || relPath == "." {
				return nil
			}
			normalized := filepath.ToSlash(relPath)
			if strings.HasPrefix(normalized, "assets") || strings.HasPrefix(normalized, "samples") {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		tmpl, err := schema.LoadTemplateFile(path)
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		normalized := filepath.ToSlash(relPath)

		entry := TemplateEntry{
			ID:                 tmpl.ID,
			Category:           deriveCategory(normalized),
			SupportedSizes:     append([]domain.PageSize(nil), tmpl.Page.SupportedSizes...),
			DefaultOrientation: tmpl.Page.DefaultOrientation,
			Source:             path,
			ImageSlots:         collectImageSlots(tmpl),
			DecorationAssets:   collectDecorationAssets(tmpl),
			Template:           tmpl,
		}
		if entry.DefaultOrientation == "" {
			entry.DefaultOrientation = domain.OrientationPortrait
		}
		entries = append(entries, entry)
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category == entries[j].Category {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Category < entries[j].Category
	})

	return entries
}

func collectImageSlots(tmpl domain.Template) []domain.Slot {
	slots := make([]domain.Slot, 0)
	for _, slot := range tmpl.Slots {
		if slot.Type == domain.SlotTypeImage {
			slots = append(slots, slot)
		}
	}
	return slots
}

func collectDecorationAssets(tmpl domain.Template) []domain.Asset {
	assets := make([]domain.Asset, 0)
	for _, asset := range tmpl.Assets {
		if asset.Kind == domain.AssetKindDecoration {
			assets = append(assets, asset)
		}
	}
	return assets
}

func deriveCategory(relPath string) string {
	if relPath == "" {
		return ""
	}

	segments := strings.Split(relPath, "/")
	if len(segments) == 0 {
		return ""
	}

	return strings.Title(segments[0])
}

func (e TemplateEntry) Label() string {
	if e.Category == "" {
		return e.ID
	}
	return e.ID + " (" + e.Category + ")"
}

func (e TemplateEntry) Description() string {
	parts := []string{
		"sizes: " + formatSizes(e.SupportedSizes),
	}
	if len(e.ImageSlots) > 0 {
		parts = append(parts, "images: "+pluralize(len(e.ImageSlots), "slot", "slots"))
	}
	if len(e.DecorationAssets) > 0 {
		parts = append(parts, "decorations: "+pluralize(len(e.DecorationAssets), "asset", "assets"))
	}
	return strings.Join(parts, " • ")
}

func (e TemplateEntry) PrimaryImageSlot() (domain.Slot, bool) {
	if len(e.ImageSlots) == 0 {
		return domain.Slot{}, false
	}

	for _, wanted := range primaryImageSlotPriority {
		for _, slot := range e.ImageSlots {
			if strings.EqualFold(slot.ID, wanted) {
				return slot, true
			}
		}
	}

	return e.ImageSlots[0], true
}

func (e TemplateEntry) AdditionalImageSlots() []domain.Slot {
	primary, ok := e.PrimaryImageSlot()
	if !ok {
		return nil
	}

	extra := make([]domain.Slot, 0, len(e.ImageSlots))
	for _, slot := range e.ImageSlots {
		if slot.ID != primary.ID {
			extra = append(extra, slot)
		}
	}

	return extra
}

func templateDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return filepath.Join(projectRoot, "templates")
}

func formatSizes(sizes []domain.PageSize) string {
	if len(sizes) == 0 {
		return "none"
	}

	values := make([]string, len(sizes))
	for i, size := range sizes {
		values[i] = string(size)
	}
	return strings.Join(values, ", ")
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}
