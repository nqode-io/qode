package config

// DefaultConfig returns a Config with sensible defaults applied.
func DefaultConfig() Config {
	return Config{
		Project: ProjectConfig{
			Topology: TopologySingle,
		},
		Review: ReviewConfig{
			MinCodeScore:     8.0,
			MinSecurityScore: 8.0,
		},
		Scoring: ScoringConfig{
			RefineTargetScore: 25,
		},
		IDE: IDEConfig{
			Cursor:     CursorIDEConfig{Enabled: true, RulesDir: ".cursorrules", CommandsDir: ".cursor/commands"},
			VSCode:     VSCodeIDEConfig{Enabled: true, Launch: true, Tasks: true, Settings: true, Extensions: true},
			ClaudeCode: ClaudeCodeIDEConfig{Enabled: true, ClaudeMD: true, SlashCommands: true},
		},
		Knowledge: KnowledgeConfig{
			AutoDiscover: true,
		},
		Architecture: ArchitectureConfig{
			DRYRules:  DRYRulesConfig{Enabled: true, MaxRepetitions: 3},
			CleanCode: CleanCodeConfig{MaxFunctionLines: 50},
		},
		Branch: BranchConfig{KeepBranchContext: false},
	}
}

// StackDefaults maps a stack name to its default test/build commands.
var StackDefaults = map[string]TestConfig{
	"react": {
		Unit:  "npm test",
		Lint:  "npm run lint",
		Build: "npm run build",
	},
	"nextjs": {
		Unit:  "npm test",
		Lint:  "npm run lint",
		Build: "npm run build",
	},
	"angular": {
		Unit:  "ng test --no-watch --code-coverage",
		E2E:   "ng e2e",
		Lint:  "ng lint",
		Build: "ng build --configuration production",
	},
	"vue": {
		Unit:  "npm test",
		Lint:  "npm run lint",
		Build: "npm run build",
	},
	"svelte": {
		Unit:  "npm test",
		Lint:  "npm run lint",
		Build: "npm run build",
	},
	"typescript": {
		Unit:  "npm test",
		Lint:  "npm run lint",
		Build: "npx tsc --noEmit",
	},
	"dotnet": {
		Unit:  "dotnet test",
		Lint:  "dotnet format --verify-no-changes",
		Build: "dotnet build",
	},
	"java": {
		Unit:  "mvn test",
		Lint:  "mvn checkstyle:check",
		Build: "mvn package -DskipTests",
	},
	"python": {
		Unit:  "pytest",
		Lint:  "ruff check .",
		Build: "",
	},
	"go": {
		Unit:  "go test ./...",
		Lint:  "golangci-lint run",
		Build: "go build ./...",
	},
}
