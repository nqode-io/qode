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
)

//go:embed templates
var embeddedFS embed.FS

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
	}
	return e, nil
}

// TemplateData is passed into every template.
type TemplateData struct {
	Project    config.ProjectConfig
	Layers     []config.LayerConfig
	Branch     string
	Ticket     string
	Notes      string
	Analysis   string
	Spec       string
	Diff       string
	Extra      string
	KB         string
	OutputPath string // when set, templates append file-write instructions
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
