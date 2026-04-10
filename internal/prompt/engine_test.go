package prompt

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestPRCreateTemplate_Conditionals(t *testing.T) {
	t.Parallel()
	e, err := NewEngine(t.TempDir())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	cases := []struct {
		name        string
		data        TemplateData
		mustContain []string
		mustAbsent  []string
	}{
		{
			name: "all-sections-present",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithTicket("UNIQUE-TICKET-SENTINEL").
				WithSpec("spec content").
				WithDiff("UNIQUE-DIFF-SENTINEL").
				WithCodeReview("UNIQUE-CODEREVIEW-SENTINEL").
				WithSecurityReview("UNIQUE-SECURITYREVIEW-SENTINEL").
				WithDraftPR(false).
				Build(),
			mustContain: []string{
				"UNIQUE-TICKET-SENTINEL",
				"UNIQUE-DIFF-SENTINEL",
				"UNIQUE-CODEREVIEW-SENTINEL",
				"UNIQUE-SECURITYREVIEW-SENTINEL",
				"already exists",
				"## Ticket context",
			},
		},
		{
			name: "ticket-absent",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithSpec("spec content").
				Build(),
			mustAbsent: []string{"## Ticket context"},
		},
		{
			name: "reviews-absent",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithSpec("spec content").
				Build(),
			mustAbsent: []string{"## Code Review", "## Security Review"},
		},
		{
			name: "draft-true",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithSpec("spec content").
				WithDraftPR(true).
				Build(),
			mustContain: []string{"as a draft"},
		},
		{
			name: "draft-false",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithSpec("spec content").
				WithDraftPR(false).
				Build(),
			mustAbsent: []string{"as a draft"},
		},
		{
			name: "pr-exists-check-always-present",
			data: NewTemplateData("proj", "feat").
				WithBranchDir("/tmp/branch").
				WithBaseBranch("main").
				WithSpec("spec content").
				Build(),
			mustContain: []string{"already exists"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.Render("pr/create", tc.data)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			for _, want := range tc.mustContain {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q", want)
				}
			}
			for _, absent := range tc.mustAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("expected output NOT to contain %q", absent)
				}
			}
		})
	}
}

func TestNewEngine(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("/tmp/my-project")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if got := e.ProjectName(); got != "my-project" {
		t.Errorf("ProjectName() = %q, want %q", got, "my-project")
	}
}

func TestRender_EmbeddedTemplate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := NewTemplateData(e.ProjectName(), "main").Build()
	out, err := e.Render("refine/base", data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output from embedded template")
	}
}

func TestRender_LocalOverride(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	overridePath := filepath.Join(root, ".qode", "prompts", "refine", "base.md.tmpl")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte("LOCAL:{{.Project.Name}}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := NewTemplateData("test-proj", "main").Build()
	out, err := e.Render("refine/base", data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "LOCAL:test-proj" {
		t.Errorf("expected local override output %q, got %q", "LOCAL:test-proj", out)
	}
}

func TestRender_MissingTemplate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := minimalTemplateData().Build()
	_, err = e.Render("does/not/exist", data)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRender_MalformedTemplate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	overridePath := filepath.Join(root, ".qode", "prompts", "bad.md.tmpl")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte("{{.Bad.Path.Here}}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := minimalTemplateData().Build()
	_, err = e.Render("bad", data)
	if err == nil {
		t.Fatal("expected error for malformed template")
	}
}

func TestRender_FuncMap_Join(t *testing.T) {
	t.Parallel()
	e, err := NewEngine(t.TempDir())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	tests := []struct {
		name  string
		sep   string
		items []string
		want  string
	}{
		{"comma sep", ", ", []string{"a", "b", "c"}, "a, b, c"},
		{"empty sep", "", []string{"a", "b"}, "ab"},
		{"single item", ", ", []string{"only"}, "only"},
		{"empty slice", ", ", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpl, err := template.New("join").Funcs(e.funcMap).Parse(`{{join .Sep .Items}}`)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, map[string]any{"Sep": tt.sep, "Items": tt.items}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRender_FuncMap_Add(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	overridePath := filepath.Join(root, ".qode", "prompts", "test-add.md.tmpl")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte(`{{add .TargetScore 5}}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := minimalTemplateData().WithTargetScore(10).Build()
	out, err := e.Render("test-add", data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "15" {
		t.Errorf("add(10,5) = %q, want %q", out, "15")
	}
}

func TestRender_FuncMap_Pct(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	overridePath := filepath.Join(root, ".qode", "prompts", "test-pct.md.tmpl")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte(`{{printf "%.1f" (pct 75.0 .TargetScore)}}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	e, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	data := minimalTemplateData().WithTargetScore(10).Build()
	out, err := e.Render("test-pct", data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "7.5" {
		t.Errorf("pct(75.0, 10) = %q, want %q", out, "7.5")
	}
}

func TestEmbeddedTemplates(t *testing.T) {
	t.Parallel()
	templates, err := EmbeddedTemplates()
	if err != nil {
		t.Fatalf("EmbeddedTemplates: %v", err)
	}

	expected := []string{
		"refine/base",
		"spec/base",
		"start/base",
		"review/code",
		"review/security",
		"scoring/judge_refine",
		"knowledge/add-branch",
		"knowledge/add-context",
	}

	for _, name := range expected {
		content, ok := templates[name]
		if !ok {
			t.Errorf("missing expected template %q", name)
			continue
		}
		if len(content) == 0 {
			t.Errorf("template %q has empty content", name)
		}
	}

	// Verify no .md.tmpl suffix leaked into keys.
	for name := range templates {
		if strings.HasSuffix(name, ".md.tmpl") {
			t.Errorf("template key %q should not have .md.tmpl suffix", name)
		}
	}
}
