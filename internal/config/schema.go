package config

// Config is the root configuration loaded from qode.yaml.
type Config struct {
	QodeVersion  string             `yaml:"qode_version,omitempty"`
	TicketSystem TicketSystemConfig `yaml:"ticket_system,omitempty"`
	Review       ReviewConfig       `yaml:"review,omitempty"`
	Scoring      ScoringConfig      `yaml:"scoring,omitempty"`
	IDE          IDEConfig          `yaml:"ide,omitempty"`
	Knowledge    KnowledgeConfig    `yaml:"knowledge,omitempty"`
	Branch       BranchConfig       `yaml:"branch,omitempty"`
}

// TicketSystemConfig describes the external ticket system integration.
type TicketSystemConfig struct {
	Type       string     `yaml:"type,omitempty"` // jira, azure-devops, linear, github, notion, manual
	URL        string     `yaml:"url,omitempty"`
	ProjectKey string     `yaml:"project_key,omitempty"`
	Auth       AuthConfig `yaml:"auth,omitempty"`
}

// AuthConfig holds authentication details for external integrations.
type AuthConfig struct {
	Method string `yaml:"method,omitempty"` // token, oauth, pat
	EnvVar string `yaml:"env_var,omitempty"`
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
	Strict      bool                    `yaml:"strict,omitempty"`
	Rubrics     map[string]RubricConfig `yaml:"rubrics,omitempty"`
}

// IDEConfig controls which IDE integrations are generated.
type IDEConfig struct {
	Cursor     CursorIDEConfig     `yaml:"cursor,omitempty"`
	ClaudeCode ClaudeCodeIDEConfig `yaml:"claude_code,omitempty"`
}

// CursorIDEConfig controls Cursor IDE integration.
type CursorIDEConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// ClaudeCodeIDEConfig controls Claude Code integration.
type ClaudeCodeIDEConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// KnowledgeConfig controls the knowledge base.
type KnowledgeConfig struct {
	Path string `yaml:"path,omitempty"`
}

// BranchConfig controls branch lifecycle behaviour.
type BranchConfig struct {
	KeepBranchContext bool `yaml:"keep_branch_context,omitempty"`
}

// ScoringFileConfig is written to and read from .qode/scoring.yaml.
// It holds only the rubric definitions, keeping them separate from qode.yaml
// so that re-running qode init never overwrites user-customised rubrics.
type ScoringFileConfig struct {
	Rubrics map[string]RubricConfig `yaml:"rubrics,omitempty"`
}
