package cli

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaeyoung0509/letterpress/internal/schema"
)

const defaultTemplateDir = "templates"

type templateEntry struct {
	ID   string
	Path string
}

// NewTemplatesCmd returns the command group for template-related utilities.
func NewTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Work with shipped templates",
	}
	cmd.AddCommand(newTemplatesListCmd())
	return cmd
}

func newTemplatesListCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := dir
			if targetDir == "" {
				targetDir = defaultTemplateDir
			}
			return runTemplatesList(cmd.OutOrStdout(), targetDir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultTemplateDir, "template root directory")
	return cmd
}

func runTemplatesList(out io.Writer, dir string) error {
	entries, err := scanTemplates(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "no templates found")
		return nil
	}
	for _, entry := range entries {
		fmt.Fprintf(out, "%s\t%s\n", entry.ID, entry.Path)
	}
	return nil
}

func scanTemplates(dir string) ([]templateEntry, error) {
	var entries []templateEntry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "assets", "samples":
				return fs.SkipDir
			}
			return nil
		}
		if !isTemplateFile(path) {
			return nil
		}
		tmpl, err := schema.LoadTemplateFile(path)
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}
		entries = append(entries, templateEntry{ID: tmpl.ID, Path: path})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

func isTemplateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
