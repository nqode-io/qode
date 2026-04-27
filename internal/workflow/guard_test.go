package workflow

import (
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/qodecontext"
)

func TestCheckStep(t *testing.T) {
	t.Parallel()
	defaultCfg := config.DefaultConfig()
	customCfg := config.DefaultConfig()
	customCfg.Scoring.TargetScore = 20

	cases := []struct {
		name        string
		step        string
		ctx         *qodecontext.Context
		cfg         *config.Config
		wantBlocked bool
		wantMsg     string // substring that must appear in Message when blocked
	}{
		{
			name:        "spec/no-analysis",
			step:        "spec",
			ctx:         &qodecontext.Context{},
			cfg:         &defaultCfg,
			wantBlocked: true,
			wantMsg:     "refined-analysis.md",
		},
		{
			name: "spec/unscored",
			step: "spec",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "some content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 0}},
			},
			cfg:         &defaultCfg,
			wantBlocked: true,
			wantMsg:     "qode-plan-judge",
		},
		{
			name: "spec/below-default-min",
			step: "spec",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "some content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 20}},
			},
			cfg:         &defaultCfg,
			wantBlocked: true,
			wantMsg:     "qode-plan-refine",
		},
		{
			name: "spec/meets-default-min",
			step: "spec",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "some content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 25}},
			},
			cfg:         &defaultCfg,
			wantBlocked: false,
		},
		{
			name: "spec/custom-target-met",
			step: "spec",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "some content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 20}},
			},
			cfg:         &customCfg,
			wantBlocked: false,
		},
		{
			name: "spec/custom-target-not-met",
			step: "spec",
			ctx: &qodecontext.Context{
				RefinedAnalysis: "some content",
				Iterations:      []qodecontext.Iteration{{Number: 1, Score: 19}},
			},
			cfg:         &customCfg,
			wantBlocked: true,
			wantMsg:     "qode-plan-refine",
		},
		{
			name:        "start/no-spec",
			step:        "start",
			ctx:         &qodecontext.Context{},
			cfg:         &defaultCfg,
			wantBlocked: true,
			wantMsg:     "spec.md",
		},
		{
			name:        "start/spec-present",
			step:        "start",
			ctx:         &qodecontext.Context{Spec: "spec content"},
			cfg:         &defaultCfg,
			wantBlocked: false,
		},
		{
			name:        "review-code/always-passes",
			step:        "review-code",
			ctx:         &qodecontext.Context{},
			cfg:         &defaultCfg,
			wantBlocked: false,
		},
		{
			name:        "review-security/always-passes",
			step:        "review-security",
			ctx:         &qodecontext.Context{},
			cfg:         &defaultCfg,
			wantBlocked: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := CheckStep(tc.step, tc.ctx, tc.cfg)
			if result.Blocked != tc.wantBlocked {
				t.Errorf("Blocked: want %v, got %v (message: %q)", tc.wantBlocked, result.Blocked, result.Message)
			}
			if tc.wantBlocked && tc.wantMsg != "" && !strings.Contains(result.Message, tc.wantMsg) {
				t.Errorf("Message: want substring %q, got %q", tc.wantMsg, result.Message)
			}
		})
	}
}
