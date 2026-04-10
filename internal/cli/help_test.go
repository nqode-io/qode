package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/config"
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

func TestBuildStatusLines_AllEmpty(t *testing.T) {
	ctx := &branchcontext.Context{}
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
	ctx := &branchcontext.Context{Ticket: "JIRA-123"}
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

	ctx := &branchcontext.Context{
		Ticket:          "JIRA-123",
		RefinedAnalysis: "analysis",
		Spec:            "spec content",
		ContextDir:      ctxDir,
		PRURL:           "https://github.com/org/repo/pull/1",
		Iterations: []branchcontext.Iteration{
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
}

func TestRefineStatus_Table(t *testing.T) {
	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	tests := []struct {
		name     string
		ctx      *branchcontext.Context
		wantSub  string
		wantNext string
	}{
		{
			name:     "no analysis",
			ctx:      &branchcontext.Context{},
			wantSub:  "Not started",
			wantNext: "refine",
		},
		{
			name: "unscored",
			ctx: &branchcontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []branchcontext.Iteration{{Number: 1, Score: 0}},
			},
			wantSub:  "unscored",
			wantNext: "judge",
		},
		{
			name: "below minimum",
			ctx: &branchcontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []branchcontext.Iteration{{Number: 1, Score: 10}},
			},
			wantSub:  "below minimum",
			wantNext: "below minimum",
		},
		{
			name: "passing",
			ctx: &branchcontext.Context{
				RefinedAnalysis: "content",
				Iterations:      []branchcontext.Iteration{{Number: 1, Score: 25}},
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
			status := reviewItemStatus(tt.present, tt.score, tt.min, "/qode-review-code", tt.maxScore, &upNext)
			if !strings.Contains(status, tt.wantSub) {
				t.Errorf("status = %q, want substring %q", status, tt.wantSub)
			}
		})
	}
}
