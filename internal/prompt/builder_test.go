package prompt

import (
	"testing"

	"github.com/nqode/qode/internal/scoring"
)

func TestNewTemplateData_Defaults(t *testing.T) {
	t.Parallel()
	data := NewTemplateData("my-project", "feat/login").Build()

	if data.Project.Name != "my-project" {
		t.Errorf("Project.Name = %q, want %q", data.Project.Name, "my-project")
	}
	if data.Branch != "feat/login" {
		t.Errorf("Branch = %q, want %q", data.Branch, "feat/login")
	}
	if data.IDE != "" {
		t.Errorf("IDE = %q, want empty", data.IDE)
	}
	if data.Ticket != "" {
		t.Errorf("Ticket = %q, want empty", data.Ticket)
	}
	if data.TargetScore != 0 {
		t.Errorf("TargetScore = %d, want 0", data.TargetScore)
	}
}

func TestTemplateDataBuilder_AllSetters(t *testing.T) {
	t.Parallel()
	rubric := scoring.Rubric{
		Dimensions: []scoring.Dimension{
			{Name: "clarity", Weight: 5},
		},
	}

	data := NewTemplateData("proj", "main").
		WithIDE("claude").
		WithOutputPath("/tmp/out.md").
		WithBranchDir("/root/.qode/branches/main").
		WithRubric(rubric).
		WithTargetScore(25).
		WithMinPassScore(10.0).
		WithTicket("ticket content").
		WithAnalysis("analysis content").
		WithSpec("spec content").
		WithDiff("diff content").
		WithExtra("extra content").
		WithKB("kb content").
		WithLessons("lesson content").
		Build()

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"IDE", data.IDE, "claude"},
		{"OutputPath", data.OutputPath, "/tmp/out.md"},
		{"BranchDir", data.BranchDir, "/root/.qode/branches/main"},
		{"Ticket", data.Ticket, "ticket content"},
		{"Analysis", data.Analysis, "analysis content"},
		{"Spec", data.Spec, "spec content"},
		{"Diff", data.Diff, "diff content"},
		{"Extra", data.Extra, "extra content"},
		{"KB", data.KB, "kb content"},
		{"Lessons", data.Lessons, "lesson content"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
	if data.TargetScore != 25 {
		t.Errorf("TargetScore = %d, want 25", data.TargetScore)
	}
	if data.MinPassScore != 10.0 {
		t.Errorf("MinPassScore = %f, want 10.0", data.MinPassScore)
	}
	if len(data.Rubric.Dimensions) != 1 || data.Rubric.Dimensions[0].Name != "clarity" {
		t.Errorf("Rubric not set correctly: %+v", data.Rubric)
	}
}

func TestTemplateDataBuilder_PRFields(t *testing.T) {
	t.Parallel()
	data := NewTemplateData("proj", "main").
		WithBaseBranch("develop").
		WithCodeReview("code review content").
		WithSecurityReview("security review content").
		WithDraftPR(true).
		Build()

	if data.BaseBranch != "develop" {
		t.Errorf("BaseBranch = %q, want %q", data.BaseBranch, "develop")
	}
	if data.CodeReview != "code review content" {
		t.Errorf("CodeReview = %q, want %q", data.CodeReview, "code review content")
	}
	if data.SecurityReview != "security review content" {
		t.Errorf("SecurityReview = %q, want %q", data.SecurityReview, "security review content")
	}
	if !data.DraftPR {
		t.Error("DraftPR = false, want true")
	}
}
