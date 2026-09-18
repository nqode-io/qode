// Package config loads, validates, and normalizes qode.yaml configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"gopkg.in/yaml.v3"
)

// ErrConfigNotFound is returned when qode.yaml cannot be located in any parent directory.
var ErrConfigNotFound = errors.New("config not found")

const (
	// ConfigFileName is the default config file name.
	ConfigFileName = "qode.yaml"
	// QodeDir is the per-project qode state directory.
	QodeDir = ".qode"
	// ScoringFileName is the per-project scoring rubric file inside QodeDir.
	ScoringFileName = "scoring.yaml"
)

// Load reads and merges configuration from:
//  1. Default values
//  2. qode.yaml in root (FindRoot walks ancestors; Load does not)
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

	// Try to load scoring rubrics from .qode/scoring.yaml (overrides defaults).
	scoringPath := filepath.Join(root, QodeDir, ScoringFileName)
	if err := mergeScoringFromFile(scoringPath, &cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading %s: %w", scoringPath, err)
	}

	// Try to load user-level config.
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, QodeDir, "config.yaml")
		if err := mergeFromFile(userPath, &cfg); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading %s: %w", userPath, err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid qode config: %w", err)
	}

	return &cfg, nil
}

func mergeScoringFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sf ScoringFileConfig
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if sf.Rubrics != nil {
		cfg.Scoring.Rubrics = sf.Rubrics
	}
	return nil
}

// Save writes the config to qode.yaml in the given directory.
func Save(root string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(root, ConfigFileName)
	return iokit.WriteFile(path, data, 0644)
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
			return "", fmt.Errorf("no %s found in %s or any parent directory: %w", ConfigFileName, dir, ErrConfigNotFound)
		}
		abs = parent
	}
}

// Deprecation notice wording. %s is the path of the file that carried the key.
const (
	bothKeysNotice = "warning: %s: both 'agents:' and 'ide:' are set — " +
		"'agents:' wins, 'ide:' is ignored. Delete the 'ide:' block."
	legacyKeyNotice = "warning: %s: 'ide:' is deprecated — read as 'agents:'. " +
		"Rename the key to 'agents:'."
)

// LegacyKeys records deprecated configuration keys seen while loading, in load
// order: the project qode.yaml before ~/.qode/config.yaml.
type LegacyKeys struct {
	IDEKeyPaths  []string // files that carried `ide:` alone
	BothKeyPaths []string // files that carried both `ide:` and `agents:`
}

// Notices returns the user-facing lines for those observations: the both-keys
// warnings first, then the deprecation warnings, each in load order. Pure: it
// formats strings and performs no I/O, so config stays printer-free.
func (c *Config) Notices() []string {
	var out []string
	for _, p := range c.Legacy.BothKeyPaths {
		out = append(out, fmt.Sprintf(bothKeysNotice, p))
	}
	for _, p := range c.Legacy.IDEKeyPaths {
		out = append(out, fmt.Sprintf(legacyKeyNotice, p))
	}
	return out
}

// legacyProbe detects which agent keys ONE file carries. It is needed because
// mergeFromFile unmarshals every file into the same Config, so a legacy project
// file followed by a modern user file is otherwise indistinguishable from one
// file carrying both keys.
type legacyProbe struct {
	Agents *AgentsConfig `yaml:"agents"`
	IDE    *AgentsConfig `yaml:"ide"`
}

func mergeFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var probe legacyProbe
	// Bare, like the unmarshal below it: Load already names the file, and the
	// probe is the first thing to reject a mistyped agents: or ide: block, so a
	// wrapper here would stamp the path into the message twice.
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}
	return promoteLegacy(cfg, path, probe.Agents != nil)
}

// promoteLegacy applies this file's deprecated ide: block over cfg.Agents key by
// key, so an agent the block does not name keeps the value it already has. A file
// that also carries agents: is not promoted: agents: wins. The node is zeroed
// either way, so it can never survive into the next file or into Save.
func promoteLegacy(cfg *Config, path string, sawAgents bool) error {
	node := cfg.IDE
	cfg.IDE = yaml.Node{}
	if node.Kind == 0 {
		return nil
	}
	if sawAgents {
		cfg.Legacy.BothKeyPaths = append(cfg.Legacy.BothKeyPaths, path)
		return nil
	}
	cfg.Legacy.IDEKeyPaths = append(cfg.Legacy.IDEKeyPaths, path)
	if err := node.Decode(&cfg.Agents); err != nil {
		return fmt.Errorf("parsing %s: decoding deprecated 'ide:' block: %w", path, err)
	}
	return nil
}
