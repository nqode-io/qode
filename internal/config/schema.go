package config

import "gopkg.in/yaml.v3"

// Config is the root configuration loaded from qode.yaml.
type Config struct {
	QodeVersion string        `yaml:"qode_version,omitempty"`
	Review      ReviewConfig  `yaml:"review,omitempty"`
	Scoring     ScoringConfig `yaml:"scoring,omitempty"`
	Agents      AgentsConfig  `yaml:"agents,omitempty"`
	// IDE is the deprecated spelling of Agents. It is read-only: Load decodes it over
	// Agents and zeroes it, so it is never marshalled back. Declared as a yaml.Node,
	// not a *AgentsConfig, so that a partial block decodes over the seeded defaults
	// instead of over a fresh zero struct.
	IDE       yaml.Node       `yaml:"ide,omitempty"`
	Knowledge KnowledgeConfig `yaml:"knowledge,omitempty"`
	Diff      DiffConfig      `yaml:"diff,omitempty"`
}

// ReviewConfig sets thresholds for code and security reviews.
type ReviewConfig struct {
	MinCodeScore     float64 `yaml:"min_code_score,omitempty"`
	MinSecurityScore float64 `yaml:"min_security_score,omitempty"`
}

// DimensionConfig is one scoring axis defined in qode.yaml.
type DimensionConfig struct {
	Name        string   `yaml:"name"`
	Weight      int      `yaml:"weight"`
	Description string   `yaml:"description,omitempty"`
	Levels      []string `yaml:"levels,omitempty"`
}

// RubricConfig holds the dimensions for one rubric kind.
type RubricConfig struct {
	Dimensions []DimensionConfig `yaml:"dimensions"`
}

// ScoringConfig controls the scoring engine.
type ScoringConfig struct {
	TargetScore int                     `yaml:"target_score,omitempty"`
	Strict      bool                    `yaml:"strict"`
	Rubrics     map[string]RubricConfig `yaml:"rubrics,omitempty"`
}

// AgentsConfig controls which agent integrations are generated.
type AgentsConfig struct {
	Cursor     CursorAgentConfig     `yaml:"cursor,omitempty"`
	ClaudeCode ClaudeCodeAgentConfig `yaml:"claude_code,omitempty"`
	Codex      CodexAgentConfig      `yaml:"codex,omitempty"`
	OpenCode   OpenCodeAgentConfig   `yaml:"opencode,omitempty"`
}

// CursorAgentConfig controls Cursor agent integration.
type CursorAgentConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// ClaudeCodeAgentConfig controls Claude Code integration.
type ClaudeCodeAgentConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// CodexAgentConfig controls Codex agent integration.
type CodexAgentConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// OpenCodeAgentConfig controls OpenCode integration.
type OpenCodeAgentConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// KnowledgeConfig controls the knowledge base.
type KnowledgeConfig struct {
	Path string `yaml:"path,omitempty"`
}

// DiffConfig controls how the diff is generated for review prompts.
type DiffConfig struct {
	Command string `yaml:"command,omitempty"`
}

// ScoringFileConfig is written to and read from .qode/scoring.yaml.
// It holds only the rubric definitions, keeping them separate from qode.yaml
// so that re-running qode init never overwrites user-customised rubrics.
type ScoringFileConfig struct {
	Rubrics map[string]RubricConfig `yaml:"rubrics,omitempty"`
}
