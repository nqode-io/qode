package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the default config file name.
	ConfigFileName = "qode.yaml"
	// WorkspaceConfigFileName is the workspace-level config file name.
	WorkspaceConfigFileName = "qode-workspace.yaml"
	// QodeDir is the per-project qode state directory.
	QodeDir = ".qode"
)

// Load reads and merges configuration from:
//  1. Default values
//  2. qode.yaml in root (or closest ancestor)
//  3. ~/.qode/config.yaml (user-level overrides)
//
// CLI flags override all of these at call site.
func Load(root string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load project config.
	projectPath := filepath.Join(root, ConfigFileName)
	if err := mergeFromFile(projectPath, &cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading %s: %w", projectPath, err)
	}

	// Try to load user-level config.
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, QodeDir, "config.yaml")
		if err := mergeFromFile(userPath, &cfg); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading %s: %w", userPath, err)
		}
	}

	return &cfg, nil
}

// Save writes the config to qode.yaml in the given directory.
func Save(root string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(root, ConfigFileName)
	return os.WriteFile(path, data, 0644)
}

// FindRoot walks up from dir looking for qode.yaml and returns the directory
// containing it. Returns an error if no config is found.
func FindRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ConfigFileName)); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no %s found in %s or any parent directory", ConfigFileName, dir)
		}
		abs = parent
	}
}

func mergeFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}
