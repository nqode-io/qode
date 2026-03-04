package config

// Config is the root configuration loaded from qode.yaml.
type Config struct {
	Project      ProjectConfig      `yaml:"project"`
	TicketSystem TicketSystemConfig `yaml:"ticket_system,omitempty"`
	Review       ReviewConfig       `yaml:"review,omitempty"`
	Scoring      ScoringConfig      `yaml:"scoring,omitempty"`
	IDE          IDEConfig          `yaml:"ide,omitempty"`
	Workspace    WorkspaceConfig    `yaml:"workspace,omitempty"`
	Knowledge    KnowledgeConfig    `yaml:"knowledge,omitempty"`
	Architecture ArchitectureConfig `yaml:"architecture,omitempty"`
}

// ProjectConfig describes the project and its tech layers.
type ProjectConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Topology    Topology `yaml:"topology,omitempty"` // monorepo, multirepo, single

	// Composite layers — the primary way to describe a project's tech stack.
	Layers []LayerConfig `yaml:"layers,omitempty"`

	// Shorthand for single-stack projects (expanded into a single layer internally).
	Stack string     `yaml:"stack,omitempty"`
	Test  TestConfig `yaml:"test,omitempty"`
}

// LayerConfig describes one technology layer within a project.
type LayerConfig struct {
	Name  string     `yaml:"name"`
	Path  string     `yaml:"path"`
	Stack string     `yaml:"stack"` // react, angular, nextjs, dotnet, java, python, go, typescript, vue, svelte
	Test  TestConfig `yaml:"test,omitempty"`
}

// TestConfig holds the commands for running quality checks on a layer.
type TestConfig struct {
	Unit     string         `yaml:"unit,omitempty"`
	E2E      string         `yaml:"e2e,omitempty"`
	Lint     string         `yaml:"lint,omitempty"`
	Build    string         `yaml:"build,omitempty"`
	Coverage CoverageConfig `yaml:"coverage,omitempty"`
}

// CoverageConfig controls test coverage requirements.
type CoverageConfig struct {
	Enabled       bool    `yaml:"enabled,omitempty"`
	MinPercentage float64 `yaml:"min_percentage,omitempty"`
}

// Topology is the repo layout type.
type Topology string

const (
	TopologyMonorepo  Topology = "monorepo"
	TopologyMultirepo Topology = "multirepo"
	TopologySingle    Topology = "single"
)

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

// ScoringConfig controls the two-pass scoring engine.
type ScoringConfig struct {
	TwoPass             bool `yaml:"two_pass,omitempty"`
	MaxRefineIterations int  `yaml:"max_refine_iterations,omitempty"`
	RefineTargetScore   int  `yaml:"refine_target_score,omitempty"`
}

// IDEConfig controls which IDE integrations are generated.
type IDEConfig struct {
	Cursor     CursorIDEConfig     `yaml:"cursor,omitempty"`
	VSCode     VSCodeIDEConfig     `yaml:"vscode,omitempty"`
	ClaudeCode ClaudeCodeIDEConfig `yaml:"claude_code,omitempty"`
}

// CursorIDEConfig controls Cursor IDE integration.
type CursorIDEConfig struct {
	Enabled     bool   `yaml:"enabled,omitempty"`
	RulesDir    string `yaml:"rules_dir,omitempty"`
	CommandsDir string `yaml:"commands_dir,omitempty"`
}

// VSCodeIDEConfig controls VS Code integration.
type VSCodeIDEConfig struct {
	Enabled    bool `yaml:"enabled,omitempty"`
	Launch     bool `yaml:"launch,omitempty"`
	Tasks      bool `yaml:"tasks,omitempty"`
	Settings   bool `yaml:"settings,omitempty"`
	Extensions bool `yaml:"extensions,omitempty"`
}

// ClaudeCodeIDEConfig controls Claude Code integration.
type ClaudeCodeIDEConfig struct {
	Enabled       bool `yaml:"enabled,omitempty"`
	ClaudeMD      bool `yaml:"claude_md,omitempty"`
	SlashCommands bool `yaml:"slash_commands,omitempty"`
}

// WorkspaceConfig links multiple repos in a multi-repo workspace.
type WorkspaceConfig struct {
	Repos []RepoRef `yaml:"repos,omitempty"`
}

// RepoRef identifies a repo in a multi-repo workspace.
type RepoRef struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url,omitempty"`
	Path   string `yaml:"path,omitempty"`
	Branch string `yaml:"branch,omitempty"`
}

// KnowledgeConfig controls the knowledge base.
type KnowledgeConfig struct {
	AutoDiscover bool     `yaml:"auto_discover,omitempty"`
	Paths        []string `yaml:"paths,omitempty"`
}

// ArchitectureConfig enforces coding standards.
type ArchitectureConfig struct {
	DRYRules  DRYRulesConfig  `yaml:"dry_rules,omitempty"`
	CleanCode CleanCodeConfig `yaml:"clean_code,omitempty"`
}

// DRYRulesConfig controls DRY enforcement.
type DRYRulesConfig struct {
	Enabled        bool `yaml:"enabled,omitempty"`
	MaxRepetitions int  `yaml:"max_repetitions,omitempty"`
}

// CleanCodeConfig controls code quality limits.
type CleanCodeConfig struct {
	MaxFunctionLines int `yaml:"max_function_lines,omitempty"`
}

// Layers returns the effective layers for the project. If only the shorthand
// Stack field is set, it is expanded into a single-layer slice.
func (c *Config) Layers() []LayerConfig {
	if len(c.Project.Layers) > 0 {
		return c.Project.Layers
	}
	if c.Project.Stack != "" {
		return []LayerConfig{{
			Name:  "default",
			Path:  ".",
			Stack: c.Project.Stack,
			Test:  c.Project.Test,
		}}
	}
	return nil
}
