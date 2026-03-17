package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/schema"
	"github.com/jaeyoung0509/letterpress/internal/template"
)

// NewValidateCmd returns a command that validates templates and projects.
func NewValidateCmd() *cobra.Command {
	var templatePath string
	var projectPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate template and project YAML inputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd.OutOrStdout(), templatePath, projectPath)
		},
	}

	cmd.Flags().StringVar(&templatePath, "template", "", "path to a template YAML file")
	cmd.Flags().StringVar(&projectPath, "project", "", "path to a project YAML file")
	return cmd
}

func runValidate(out io.Writer, templatePath, projectPath string) error {
	if templatePath == "" && projectPath == "" {
		return fmt.Errorf("provide at least one of --template or --project")
	}

	var (
		tmpl domain.Template
		proj domain.Project
		err  error
	)

	if templatePath != "" {
		tmpl, err = schema.LoadTemplateFile(templatePath)
		if err != nil {
			return fmt.Errorf("validate template: %w", err)
		}
		fmt.Fprintf(out, "template %s: valid\n", tmpl.ID)
	}

	if projectPath != "" {
		proj, err = schema.LoadProjectFile(projectPath)
		if err != nil {
			return fmt.Errorf("validate project: %w", err)
		}
		fmt.Fprintf(out, "project %s: valid\n", proj.Template)
	}

	if templatePath != "" && projectPath != "" {
		if proj.Template != "" && tmpl.ID != "" && proj.Template != tmpl.ID {
			return fmt.Errorf("project template %s does not match template ID %s", proj.Template, tmpl.ID)
		}
		if _, err := template.Resolve(tmpl, proj); err != nil {
			return fmt.Errorf("template/project validation failed: %w", err)
		}
		fmt.Fprintf(out, "template and project resolve successfully\n")
	}

	return nil
}
