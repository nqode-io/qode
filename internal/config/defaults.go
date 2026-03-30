package config

// DefaultConfig returns a Config with sensible defaults applied.
func DefaultConfig() Config {
	return Config{
		Project: ProjectConfig{
			Topology: TopologySingle,
		},
		Review: ReviewConfig{
			MinCodeScore:     10.0,
			MinSecurityScore: 8.0,
		},
		Scoring: ScoringConfig{
			Rubrics: DefaultRubricConfigs(),
		},
		IDE: IDEConfig{
			Cursor:     CursorIDEConfig{Enabled: true},
			ClaudeCode: ClaudeCodeIDEConfig{Enabled: true},
		},
		Knowledge: KnowledgeConfig{
			Path: ".qode/knowledge",
		},
		Branch: BranchConfig{KeepBranchContext: false},
	}
}

// DefaultRubricConfigs returns the built-in rubric configurations.
// These mirror the hardcoded Default*Rubric vars in internal/scoring/rubric.go.
func DefaultRubricConfigs() map[string]RubricConfig {
	return map[string]RubricConfig{
		"refine": {
			Dimensions: []DimensionConfig{
				{
					Name:        "Problem Understanding",
					Weight:      5,
					Description: "Correct restatement, all ambiguities resolved, user need clear",
					Levels: []string{
						"5: Perfect restatement, all ambiguities resolved, user need crystal clear",
						"4: Good understanding, minor gaps",
						"3: Adequate but surface-level",
						"2: Partial understanding, significant gaps",
						"1: Mostly incorrect or too vague",
						"0: Missing or completely wrong",
					},
				},
				{
					Name:        "Technical Analysis",
					Weight:      5,
					Description: "All affected components identified, specific file/API references",
					Levels: []string{
						"5: Identifies all affected components, correct patterns, specific file/API references",
						"4: Mostly correct, minor omissions",
						"3: Covers main points, misses some details",
						"2: Shallow analysis, important components missed",
						"1: Mostly incorrect or generic",
						"0: Missing",
					},
				},
				{
					Name:        "Risk & Edge Cases",
					Weight:      5,
					Description: "Comprehensive risks, specific edge cases with mitigation",
					Levels: []string{
						"5: Comprehensive risk analysis, specific edge cases with mitigation",
						"4: Good coverage, minor gaps",
						"3: Some risks identified, misses important ones",
						"2: Superficial",
						"1: Generic or incorrect",
						"0: Missing",
					},
				},
				{
					Name:        "Completeness",
					Weight:      5,
					Description: "All acceptance criteria, implicit requirements, scope clear",
					Levels: []string{
						"5: All acceptance criteria captured, implicit requirements identified, scope clear",
						"4: Nearly complete, minor omissions",
						"3: Main requirements captured, gaps exist",
						"2: Significant gaps",
						"1: Incomplete",
						"0: Missing",
					},
				},
				{
					Name:        "Actionability",
					Weight:      5,
					Description: "Concrete tasks, correct order, prerequisites, each one commit",
					Levels: []string{
						"5: Clear concrete tasks, correct order, prerequisites identified, each task is one commit",
						"4: Mostly actionable, minor improvements possible",
						"3: Tasks defined but too coarse or unclear",
						"2: Hard to act on",
						"1: Very vague",
						"0: Missing",
					},
				},
			},
		},
		"review": {
			Dimensions: []DimensionConfig{
				{Name: "Correctness", Weight: 2, Description: "Implements spec correctly, no logic bugs"},
				{Name: "Code Quality", Weight: 2, Description: "Readable, maintainable, well-named"},
				{Name: "Architecture", Weight: 2, Description: "Follows patterns, correct separation of concerns"},
				{Name: "Error Handling", Weight: 2, Description: "All error paths handled explicitly"},
				{Name: "Testing", Weight: 2, Description: "Tests present and cover edge cases"},
				{Name: "Performance", Weight: 2, Description: "No obvious performance issues, unnecessary allocations, or blocking calls"},
			},
		},
		"security": {
			Dimensions: []DimensionConfig{
				{Name: "Injection Prevention", Weight: 2, Description: "No SQL/command/template injection vectors"},
				{Name: "Auth & AuthZ", Weight: 2, Description: "Authentication bypass and IDOR prevention"},
				{Name: "Data Exposure", Weight: 2, Description: "No PII leak, secure storage"},
				{Name: "Input Validation", Weight: 2, Description: "All inputs validated and sanitised"},
				{Name: "Dependency Safety", Weight: 2, Description: "No known CVEs in new deps"},
			},
		},
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
