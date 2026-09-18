// Package prompt renders Go templates with local-override-first resolution against embedded fallbacks.
package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/scoring"
)

//go:embed templates
var embeddedFS embed.FS

// Agent identifiers accepted by TemplateData.Agent.
const (
	agentClaude   = "claude"
	agentCursor   = "cursor"
	agentOpenCode = "opencode"
)

// TemplateProject holds the project name for template rendering.
type TemplateProject struct {
	Name string
}

// Engine renders prompt templates with project context.
type Engine struct {
	root    string
	funcMap template.FuncMap
}

// NewEngine creates an Engine for the given project root.
func NewEngine(root string) (*Engine, error) {
	e := &Engine{root: root}
	e.funcMap = template.FuncMap{
		"join": func(sep string, items []string) string {
			result := ""
			for i, s := range items {
				if i > 0 {
					result += sep
				}
				result += s
			}
			return result
		},
		"add": func(a, b int) int { return a + b },
		// pct returns percent% of n as float64. Use with printf "%.1f" in templates.
		// Example: {{printf "%.1f" (pct 75 .Rubric.Total)}} → "7.5" for a 10-pt rubric.
		"pct": func(percent float64, n int) float64 { return float64(n) * percent / 100.0 },
		// questionTool returns the name of the structured-question tool available in the
		// given agent, or "" when that agent has none. The empty return is load-bearing: it
		// is falsy in text/template, so {{if questionTool .Agent}} is the structured-question
		// branch test and {{questionTool .Agent}} interpolates the tool name. Never return a
		// non-empty placeholder such as "none" — every such conditional would invert.
		"questionTool": func(agent string) string {
			switch agent {
			case agentClaude:
				return "AskUserQuestion"
			case agentOpenCode:
				return "question"
			default:
				return ""
			}
		},
		// hasFrontmatter reports whether the given agent's command files open with a YAML
		// frontmatter block rather than a Markdown title heading.
		"hasFrontmatter": func(agent string) bool {
			return agent == agentCursor || agent == agentOpenCode
		},
	}
	return e, nil
}

// ProjectName returns the base name of the project root directory.
func (e *Engine) ProjectName() string {
	return filepath.Base(e.root)
}

// TemplateData is passed into every template.
type TemplateData struct {
	Agent        string // target agent ("claude", "cursor", "codex" or "opencode"); used by scaffold templates
	Project      TemplateProject
	Ticket       string         // inline content; set only for knowledge/add-context
	Analysis     string         // inline content; set for knowledge/add-context and scoring judge
	Spec         string         // inline content; set only for knowledge/add-context
	Diff         string         // inline content; set only for knowledge/add-context
	Extra        string         // inline content; set for knowledge/add-context and refine (reviews, notes)
	KB           string         // knowledge base file references; set for start
	Lessons      string         // existing lesson summaries for deduplication; set for knowledge/add-context
	OutputPath   string         // when set, templates append file-write instructions
	Rubric       scoring.Rubric // scoring rubric; set for judge-refine, code-review, security-review prompts
	TargetScore  int            // pass threshold for refine judge (defaults to Rubric.Total(); overridden by scoring.target_score)
	MinPassScore float64        // minimum score to pass review; set from review.min_code_score or review.min_security_score
}

// Render renders a named template (e.g. "refine/base") with the given data.
// It checks .qode/prompts/ for local overrides first, then falls back to
// embedded templates.
func (e *Engine) Render(name string, data TemplateData) (string, error) {
	tmplContent, err := e.loadTemplate(name)
	if err != nil {
		return "", fmt.Errorf("loading template %q: %w", name, err)
	}

	t, err := template.New(name).Funcs(e.funcMap).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template %q: %w", name, err)
	}
	return buf.String(), nil
}

// EmbeddedTemplates returns all built-in template names and their contents.
// Names are relative paths like "refine/base" (without the .md.tmpl suffix).
func EmbeddedTemplates() (map[string][]byte, error) {
	templates := make(map[string][]byte)
	err := fs.WalkDir(embeddedFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md.tmpl") {
			return nil
		}
		data, readErr := embeddedFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("reading embedded template %q: %w", path, readErr)
		}
		name := strings.TrimPrefix(path, "templates/")
		name = strings.TrimSuffix(name, ".md.tmpl")
		templates[name] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// loadTemplate returns the template content, preferring local overrides.
func (e *Engine) loadTemplate(name string) (string, error) {
	// Check for local override in .qode/prompts/<name>.md.tmpl
	localPath := filepath.Join(e.root, config.QodeDir, "prompts", name+".md.tmpl")
	if data, err := os.ReadFile(localPath); err == nil {
		return string(data), nil
	}

	// Fall back to embedded templates.
	embeddedPath := "templates/" + name + ".md.tmpl"
	data, err := embeddedFS.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("template %q not found (looked in %s and embedded)", name, localPath)
	}
	return string(data), nil
}
