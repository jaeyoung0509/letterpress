package schema

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"gopkg.in/yaml.v3"
)

func ParseProject(data []byte) (domain.Project, error) {
	var project domain.Project
	if err := decodeStrict("project", data, &project); err != nil {
		return domain.Project{}, err
	}
	if err := project.Validate(); err != nil {
		return domain.Project{}, fmt.Errorf("validate project YAML: %w", err)
	}
	return project, nil
}

func ParseTemplate(data []byte) (domain.Template, error) {
	var template domain.Template
	if err := decodeStrict("template", data, &template); err != nil {
		return domain.Template{}, err
	}
	if err := template.Validate(); err != nil {
		return domain.Template{}, fmt.Errorf("validate template YAML: %w", err)
	}
	return template, nil
}

func LoadProjectFile(path string) (domain.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Project{}, fmt.Errorf("read project YAML: %w", err)
	}
	return ParseProject(data)
}

func LoadTemplateFile(path string) (domain.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Template{}, fmt.Errorf("read template YAML: %w", err)
	}
	return ParseTemplate(data)
}

func decodeStrict(kind string, data []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s YAML: %w", kind, err)
	}

	return nil
}
