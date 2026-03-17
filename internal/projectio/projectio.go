package projectio

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"gopkg.in/yaml.v3"
)

// Load reads a v1 YAML project file, validates it, and returns the parsed Project.
func Load(path string) (domain.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Project{}, fmt.Errorf("read project file %q: %w", path, err)
	}

	var project domain.Project
	if err := yaml.Unmarshal(data, &project); err != nil {
		return domain.Project{}, fmt.Errorf("parse project YAML %q: %w", path, err)
	}

	if err := project.Validate(); err != nil {
		return domain.Project{}, fmt.Errorf("validate project %q: %w", path, err)
	}

	return project, nil
}

// Save validates the project and writes it as YAML to the provided path.
func Save(path string, project domain.Project) error {
	if err := project.Validate(); err != nil {
		return fmt.Errorf("validate project before save: %w", err)
	}

	data, err := yaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("serialize project: %w", err)
	}

	if err := ensureDirectory(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write project file %q: %w", path, err)
	}

	return nil
}

func ensureDirectory(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare directory %q: %w", dir, err)
	}
	return nil
}
