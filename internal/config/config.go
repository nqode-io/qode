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
	if err := mergeFromFile(configFile{path: projectPath, display: projectPath}, &cfg); err != nil && !os.IsNotExist(err) {
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
		if err := mergeFromFile(configFile{path: userPath, display: userConfigDisplayPath}, &cfg); err != nil && !os.IsNotExist(err) {
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

// Deprecation notice wording. %s is how the file that carried the key is named,
// which is the display form of its path — see configFile.
const (
	bothKeysNotice = "warning: %s: both 'agents:' and 'ide:' are set — " +
		"'agents:' wins, 'ide:' is ignored. Delete the 'ide:' block."
	legacyKeyNotice = "warning: %s: 'ide:' is deprecated — read as 'agents:'. " +
		"Rename the key to 'agents:'."
	// emptyAgentsNotice replaces legacyKeyNotice when the file also carries an
	// `agents:` key written with no value. Renaming `ide:` in that file would
	// produce two `agents:` keys, which the next 'qode init' refuses outright, so
	// the ordinary advice has to be taken in two steps here.
	emptyAgentsNotice = "warning: %s: 'ide:' is deprecated — read as 'agents:'. " +
		"Delete the empty 'agents:' line first, then rename 'ide:' to 'agents:'."
)

// userConfigDisplayPath is how a notice names the machine-local config. Its
// absolute form carries the user's OS account name, and Upgrade never migrates that
// file, so the absolute path would be printed on every command for as long as the
// deprecated key is there. The tilde form is what the user would type anyway.
const userConfigDisplayPath = "~/" + QodeDir + "/config.yaml"

// configFile is one file Load merges: the path it reads, and the form a notice
// names it by. The two are the same for the project file, which the user acts on by
// path; they differ for the machine-local one, see userConfigDisplayPath. Errors
// still name the real path, because a load that fails has to say which file failed.
type configFile struct {
	path    string
	display string
}

// LegacyKeys records deprecated configuration keys seen while loading, as the
// display form of each file's path rather than the path Load read. Within
// each field the files are in load order, the project qode.yaml before
// ~/.qode/config.yaml; across fields they are not, because Notices drains the
// fields in turn. Every line names its own file, so the order is presentational.
type LegacyKeys struct {
	IDEKeyPaths      []string // files that carried `ide:` alone
	BothKeyPaths     []string // files that carried both `ide:` and `agents:`
	EmptyAgentsPaths []string // files that carried `ide:` beside a valueless `agents:`
}

// Notices returns the user-facing lines for those observations: the both-keys
// warnings first, then the deprecation warnings, each in load order within its
// own group. A file with
// a valueless `agents:` gets the two-step deprecation wording instead of the
// ordinary one, never both. Pure: it formats strings and performs no I/O, so
// config stays printer-free.
func (c *Config) Notices() []string {
	var out []string
	for _, p := range c.Legacy.BothKeyPaths {
		out = append(out, fmt.Sprintf(bothKeysNotice, iokit.DisplayPath(p)))
	}
	for _, p := range c.Legacy.IDEKeyPaths {
		out = append(out, fmt.Sprintf(legacyKeyNotice, iokit.DisplayPath(p)))
	}
	for _, p := range c.Legacy.EmptyAgentsPaths {
		out = append(out, fmt.Sprintf(emptyAgentsNotice, iokit.DisplayPath(p)))
	}
	return out
}

// legacyProbe detects which agent keys ONE file carries. It is needed because
// mergeFromFile unmarshals every file into the same Config, so a legacy project
// file followed by a modern user file is otherwise indistinguishable from one
// file carrying both keys.
type legacyProbe struct {
	// Agents is a node, and a value rather than a pointer, so that `agents:`
	// written with nothing after it can be told apart from no `agents:` key at
	// all: the first is a null node, the second is a node yaml never touched.
	// The two need different advice. A *yaml.Node cannot make that distinction —
	// yaml.v3 leaves a pointer nil for a null value. The block is still
	// type-checked, by the full unmarshal that follows this probe.
	Agents yaml.Node     `yaml:"agents"`
	IDE    *AgentsConfig `yaml:"ide"`
}

// readConfigFile reads one configuration file under the containment Upgrade
// already applies on the write side: a regular file, no larger than maxConfigBytes.
// Load runs on every command except init, so this is the read a hostile file
// reaches first, and it has to be refused before it is parsed: yaml.v3's
// duplicate-key check is quadratic in a mapping's key count, which is what makes a
// megabyte of valid YAML cost tens of seconds. Neither message names the path,
// because Load already names the file it was loading.
func readConfigFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	switch {
	case !info.Mode().IsRegular():
		return nil, errors.New("not a regular file")
	case info.Size() > maxConfigBytes:
		return nil, fmt.Errorf("%d bytes, past the %d-byte limit for a configuration",
			info.Size(), maxConfigBytes)
	}
	return os.ReadFile(path)
}

func mergeFromFile(f configFile, cfg *Config) error {
	data, err := readConfigFile(f.path)
	if err != nil {
		return err
	}
	// Parse the bytes once and decode that tree twice. The probe and the config
	// read the same document, and a second yaml.Unmarshal would re-parse the file
	// rather than re-walk what the first parse already built. A file with nothing
	// to carry a value — empty, or only comments — leaves the node untouched, and
	// yaml.v3 decodes such a node as a null, which is a no-op for both destinations:
	// the defaults survive, exactly as they did when this read two bare unmarshals.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	var probe legacyProbe
	// Bare, like the decode below it: Load already names the file, and the
	// probe is the first thing to reject a mistyped ide: block, so a wrapper
	// here would stamp the path into the message twice. A mistyped agents:
	// block is rejected by the decode below instead, the probe's own agents
	// field being a node that accepts any shape.
	if err := doc.Decode(&probe); err != nil {
		return err
	}
	if err := doc.Decode(cfg); err != nil {
		return err
	}
	return promoteLegacy(cfg, f, &probe.Agents)
}

// promoteLegacy applies this file's deprecated ide: block over cfg.Agents key by
// key, so an agent the block does not name keeps the value it already has. A file
// that also carries agents: with a value is not promoted: agents: wins. The node
// is zeroed either way, so it can never survive into the next file or into Save.
// agentsKey is this file's agents: value node, left at Kind 0 when the file has
// no such key. A notice names f the way the user should read it, an error names the
// file that actually failed.
//
// "Carries agents:" means something narrower here than on the write side, and the
// difference is deliberate. A null agents: counts as unset here, so the ide: block
// is still promoted and the user's values are read correctly; renameLegacyAgentsKey
// counts that same key as present and refuses to rename, because renaming would
// duplicate it. Such a file is therefore never migrated and warns on every run
// until the empty agents: key is deleted by hand — which is why it gets its own
// notice, the ordinary "rename the key" advice being the one thing that does not
// work there.
func promoteLegacy(cfg *Config, f configFile, agentsKey *yaml.Node) error {
	node := cfg.IDE
	cfg.IDE = yaml.Node{}
	if node.Kind == 0 {
		return nil
	}
	switch {
	case agentsKey.Kind != 0 && !isNull(agentsKey):
		cfg.Legacy.BothKeyPaths = append(cfg.Legacy.BothKeyPaths, f.display)
		return nil
	case agentsKey.Kind != 0:
		cfg.Legacy.EmptyAgentsPaths = append(cfg.Legacy.EmptyAgentsPaths, f.display)
	default:
		cfg.Legacy.IDEKeyPaths = append(cfg.Legacy.IDEKeyPaths, f.display)
	}
	if err := node.Decode(&cfg.Agents); err != nil {
		return fmt.Errorf("parsing %s: decoding deprecated 'ide:' block: %w", f.path, err)
	}
	return nil
}
