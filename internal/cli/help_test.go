package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/qodecontext"
)

func TestRunWorkflow(t *testing.T) {
	var buf bytes.Buffer
	if err := runWorkflow(&buf); err != nil {
		t.Fatalf("runWorkflow: %v", err)
	}
	if buf.String() != workflowList {
		t.Errorf("output does not match workflowList constant")
	}
}

func TestWorkflowList_ListsNoteAddAsOptionalHelper(t *testing.T) {
	if !strings.Contains(workflowList, "Optional helper: qode-note-add") {
		t.Fatalf("workflowList must mention qode-note-add as an optional helper, got:\n%s", workflowList)
	}
	if strings.Contains(workflowList, "3.  qode-note-add") {
		t.Fatalf("workflowList must not promote qode-note-add to a numbered workflow step, got:\n%s", workflowList)
	}
}

func TestWorkflowList_ListsKnowledgeAddContextAsOptionalHelper(t *testing.T) {
	if !strings.Contains(workflowList, "Optional helper: qode-knowledge-add-context") {
		t.Fatalf("workflowList must mention qode-knowledge-add-context as an optional helper, got:\n%s", workflowList)
	}
	numberedKnowledge := regexp.MustCompile(`(?m)^[0-9]{1,2}\.\s+qode-knowledge-add-context\b`)
	if numberedKnowledge.MatchString(workflowList) {
		t.Fatalf("workflowList must not promote qode-knowledge-add-context to a numbered workflow step, got:\n%s", workflowList)
	}
	titleLine := regexp.MustCompile(`(?m)^[0-9]{1,2}\.\s+Capture lessons learned\b`)
	if titleLine.MatchString(workflowList) {
		t.Fatalf("workflowList must not have a numbered 'Capture lessons learned' heading, got:\n%s", workflowList)
	}
}

func TestWorkflowList_SplitsReviewsIntoTwoNumberedSteps(t *testing.T) {
	codeReview := regexp.MustCompile(`(?m)^8\.\s+Code review\b`)
	if !codeReview.MatchString(workflowList) {
		t.Fatalf("workflowList must list code review as numbered step 8, got:\n%s", workflowList)
	}
	securityReview := regexp.MustCompile(`(?m)^9\.\s+Security review\b`)
	if !securityReview.MatchString(workflowList) {
		t.Fatalf("workflowList must list security review as numbered step 9, got:\n%s", workflowList)
	}
	if !strings.Contains(workflowList, "qode-review-code") || !strings.Contains(workflowList, "qode-review-security") {
		t.Fatalf("workflowList must reference both review commands, got:\n%s", workflowList)
	}
}

func TestBuildStatusLines_AllEmpty(t *testing.T) {
	ctx := &qodecontext.Context{}
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	lines, upNext := buildStatusLines(ctx, cfg, "")

	if len(lines) == 0 {
		t.Fatal("expected non-empty status lines")
	}

	// Step 1 is always completed.
	if !strings.Contains(lines[0], "Completed") {
		t.Errorf("step 1 should be completed, got: %q", lines[0])
	}
	// Step 2: no ticket.
	if !strings.Contains(lines[1], "Not started") {
		t.Errorf("step 2 should be not started, got: %q", lines[1])
	}
	// Step 3: no analysis.
	if !strings.Contains(lines[2], "Not started") {
		t.Errorf("step 3 should be not started, got: %q", lines[2])
	}
	// Step 4: no spec.
	if !strings.Contains(lines[3], "Not started") {
		t.Errorf("step 4 should be not started, got: %q", lines[3])
	}
	// Step 5: no diff.
	if !strings.Contains(lines[4], "Not started") {
		t.Errorf("step 5 should be not started, got: %q", lines[4])
	}
	// upNext should point to ticket fetch.
	if !strings.Contains(upNext, "ticket") {
		t.Errorf("upNext should mention ticket, got: %q", upNext)
	}
}

func TestBuildStatusLines_TicketOnly(t *testing.T) {
	ctx := &qodecontext.Context{Ticket: "JIRA-123"}
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	lines, upNext := buildStatusLines(ctx, cfg, "")

	// Step 2 should be completed.
	if !strings.Contains(lines[1], "Completed") {
		t.Errorf("step 2 should be completed with ticket, got: %q", lines[1])
	}
	// upNext should suggest refine since analysis is missing.
	if !strings.Contains(upNext, "refine") {
		t.Errorf("upNext should mention refine, got: %q", upNext)
	}
}

func TestBuildStatusLines_FullyComplete(t *testing.T) {
	ctxDir := t.TempDir()

	// Create review files so HasCodeReview/HasSecurityReview return true.
	for _, name := range []string{"code-review.md", "security-review.md"} {
		content := "# Review\n\nTotal Score: 11.0/12\n\nReview content."
		if err := os.WriteFile(filepath.Join(ctxDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	ctx := &qodecontext.Context{
		Ticket:          "JIRA-123",
		RefinedAnalysis: "analysis",
		Spec:            "spec content",
		ContextDir:      ctxDir,
		Iterations: []qodecontext.Iteration{
			{Number: 1, Score: 25},
		},
	}
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	lines, upNext := buildStatusLines(ctx, cfg, "some diff content")

	// All steps should be completed.
	if !strings.Contains(lines[0], "Completed") {
		t.Errorf("step 1 not completed: %q", lines[0])
	}
	if !strings.Contains(lines[1], "Completed") {
		t.Errorf("step 2 not completed: %q", lines[1])
	}
	if !strings.Contains(lines[3], "Completed") {
		t.Errorf("step 4 not completed: %q", lines[3])
	}
	if !strings.Contains(lines[4], "Completed") {
		t.Errorf("step 5 not completed: %q", lines[4])
	}
	// upNext should be empty when everything is done.
	if upNext != "" {
		t.Errorf("expected empty upNext for fully completed workflow, got: %q", upNext)
	}

	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "qode-pr-resolve") {
		t.Error("expected a status line referencing qode-pr-resolve")
	}

	codeReview := regexp.MustCompile(`(?m)^8\.\s+Code review\b`)
	if !codeReview.MatchString(joined) {
		t.Errorf("expected step 8 'Code review' line, got:\n%s", joined)
	}
	securityReview := regexp.MustCompile(`(?m)^9\.\s+Security review\b`)
	if !securityReview.MatchString(joined) {
		t.Errorf("expected step 9 'Security review' line, got:\n%s", joined)
	}

	if !strings.Contains(joined, "Optional helper: Capture lessons learned") {
		t.Errorf("expected unnumbered 'Optional helper: Capture lessons learned' line, got:\n%s", joined)
	}
	numberedLessons := regexp.MustCompile(`(?m)^[0-9]{1,2}\.\s+Capture lessons learned\b`)
	if numberedLessons.MatchString(joined) {
		t.Errorf("lessons-learned must not be numbered in status output, got:\n%s", joined)
	}
}

func TestBuildStatusLines_SeparatesReviews(t *testing.T) {
	ctx := &qodecontext.Context{
		Ticket:          "JIRA-123",
		RefinedAnalysis: "analysis",
		Spec:            "spec content",
		Iterations: []qodecontext.Iteration{
			{Number: 1, Score: 25},
		},
	}
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	lines, upNext := buildStatusLines(ctx, cfg, "diff content")
	joined := strings.Join(lines, "\n")

	codeLine := regexp.MustCompile(`(?m)^8\.\s+Code review\s+-\s+Not started\.`)
	if !codeLine.MatchString(joined) {
		t.Errorf("expected step 8 code review pending line, got:\n%s", joined)
	}
	secLine := regexp.MustCompile(`(?m)^9\.\s+Security review\s+-\s+Not started\.`)
	if !secLine.MatchString(joined) {
		t.Errorf("expected step 9 security review pending line, got:\n%s", joined)
	}

	// upNext must point at code review first when both are pending.
	if !strings.Contains(upNext, "qode-review-code") {
		t.Errorf("upNext should suggest qode-review-code first, got: %q", upNext)
	}
	if strings.Contains(upNext, "qode-review-security") {
		t.Errorf("upNext should not suggest qode-review-security while code review is pending, got: %q", upNext)
	}
}

func TestRefineStatus_Table(t *testing.T) {
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	tests := []struct {
		name     string
		ctx      *qodecontext.Context
		wantSub  string
		wantNext string
	}{
		{
			name:     "no analysis",
			ctx:      &qodecontext.Context{},
			wantSub:  "Not started",
			wantNext: "refine",
		},
		{
			name: "unscored",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 0}},
			},
			wantSub:  "unscored",
			wantNext: "judge",
		},
		{
			name: "below minimum",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 10}},
			},
			wantSub:  "below minimum",
			wantNext: "below minimum",
		},
		{
			name: "passing",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 25}},
			},
			wantSub:  "latest score: 25/",
			wantNext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upNext := ""
			status := refineStatus(tt.ctx, cfg, &upNext)
			if !strings.Contains(status, tt.wantSub) {
				t.Errorf("status = %q, want substring %q", status, tt.wantSub)
			}
			if tt.wantNext == "" && upNext != "" {
				t.Errorf("expected empty upNext, got: %q", upNext)
			}
			if tt.wantNext != "" && !strings.Contains(upNext, tt.wantNext) {
				t.Errorf("upNext = %q, want substring %q", upNext, tt.wantNext)
			}
		})
	}
}

func TestReviewItemStatus_Table(t *testing.T) {
	tests := []struct {
		name     string
		present  bool
		score    float64
		min      float64
		maxScore int
		wantSub  string
	}{
		{"not started", false, 0, 10.0, 12, "Not started"},
		{"below minimum", true, 7.5, 10.0, 12, "below minimum"},
		{"passing", true, 10.5, 10.0, 12, "Passed with score"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upNext := ""
			status := reviewItemStatus(tt.present, tt.score, tt.min, "qode-review-code", tt.maxScore, &upNext)
			if !strings.Contains(status, tt.wantSub) {
				t.Errorf("status = %q, want substring %q", status, tt.wantSub)
			}
		})
	}
}
