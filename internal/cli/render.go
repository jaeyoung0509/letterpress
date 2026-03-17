package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	exportpkg "github.com/jaeyoung0509/letterpress/internal/export"
	"github.com/jaeyoung0509/letterpress/internal/projectio"
	"github.com/jaeyoung0509/letterpress/internal/schema"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

// NewRenderCmd returns a non-interactive render command.
func NewRenderCmd() *cobra.Command {
	var (
		templatePath string
		projectPath  string
		outPath      string
		formatFlag   string
	)

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a composition without launching the TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(cmd.OutOrStdout(), templatePath, projectPath, formatFlag, outPath)
		},
	}

	cmd.Flags().StringVar(&templatePath, "template", "", "path to a template YAML file")
	cmd.Flags().StringVar(&projectPath, "project", "", "path to a project YAML file")
	cmd.Flags().StringVar(&formatFlag, "format", "", "override output format (pdf or png)")
	cmd.Flags().StringVar(&outPath, "out", "", "override output path")

	return cmd
}

func runRender(out io.Writer, templatePath, projectPath, formatFlag, outPath string) error {
	if strings.TrimSpace(templatePath) == "" {
		return fmt.Errorf("template path is required")
	}
	if strings.TrimSpace(projectPath) == "" {
		return fmt.Errorf("project path is required")
	}

	tmpl, err := schema.LoadTemplateFile(templatePath)
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}

	project, err := projectio.Load(projectPath)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}

	resolved, err := templatepkg.Resolve(tmpl, project)
	if err != nil {
		return fmt.Errorf("resolve template and project: %w", err)
	}

	format, target, err := resolveRenderTarget(project, formatFlag, outPath)
	if err != nil {
		return err
	}

	written, err := exportpkg.ComposeAndWrite(resolved, exportpkg.Options{
		Format:      format,
		Out:         target,
		Decorations: project.Options.Decorations,
	})
	if err != nil {
		return fmt.Errorf("render export: %w", err)
	}

	fmt.Fprintf(out, "rendered %s to %s\n", tmpl.ID, written)
	return nil
}

func resolveRenderTarget(project domain.Project, formatFlag, outPath string) (domain.ExportFormat, string, error) {
	target := strings.TrimSpace(outPath)
	if target == "" {
		target = strings.TrimSpace(project.Export.Out)
	}
	if target == "" {
		return "", "", fmt.Errorf("output path is required via --out or project export settings")
	}

	format := domain.ExportFormat(strings.ToLower(strings.TrimSpace(formatFlag)))
	if format == "" {
		format = project.Export.Format
	}
	if format == "" {
		format = inferFormatFromPath(target)
	}
	if format != domain.ExportFormatPDF && format != domain.ExportFormatPNG {
		return "", "", fmt.Errorf("render format must be pdf or png")
	}

	ext := strings.ToLower(filepath.Ext(target))
	if ext != "" && ext != "."+string(format) {
		return "", "", fmt.Errorf("output path %q does not match format %s", target, format)
	}

	return format, target, nil
}

func inferFormatFromPath(path string) domain.ExportFormat {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return domain.ExportFormatPDF
	case ".png":
		return domain.ExportFormatPNG
	default:
		return ""
	}
}
