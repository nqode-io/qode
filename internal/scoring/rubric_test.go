package scoring

import (
	"testing"

	"github.com/nqode/qode/internal/config"
)

func TestRubricTotal(t *testing.T) {
	r := Rubric{
		Dimensions: []Dimension{
			{Name: "A", Weight: 3},
			{Name: "B", Weight: 2},
			{Name: "C", Weight: 5},
		},
	}
	if got := r.Total(); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestRubricTotal_Empty(t *testing.T) {
	r := Rubric{}
	if got := r.Total(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestBuildRubric_NoOverride(t *testing.T) {
	rubric := BuildRubric(RubricRefine, nil)
	if rubric.Kind != RubricRefine {
		t.Errorf("expected RubricRefine, got %s", rubric.Kind)
	}
	if len(rubric.Dimensions) != len(DefaultRefineRubric.Dimensions) {
		t.Errorf("expected %d dims, got %d", len(DefaultRefineRubric.Dimensions), len(rubric.Dimensions))
	}
	if rubric.Total() != 25 {
		t.Errorf("expected total 25, got %d", rubric.Total())
	}
}

func TestBuildRubric_NilRubricsMap(t *testing.T) {
	cfg := &config.Config{} // Scoring.Rubrics is nil
	rubric := BuildRubric(RubricRefine, cfg)
	if rubric.Total() != 25 {
		t.Errorf("expected default total 25, got %d", rubric.Total())
	}
}

func TestBuildRubric_EmptyDimensions(t *testing.T) {
	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			Rubrics: map[string]config.RubricConfig{
				"refine": {Dimensions: []config.DimensionConfig{}},
			},
		},
	}
	rubric := BuildRubric(RubricRefine, cfg)
	// Empty dimensions → falls back to default
	if rubric.Total() != 25 {
		t.Errorf("expected fallback total 25, got %d", rubric.Total())
	}
}

func TestBuildRubric_WithOverride(t *testing.T) {
	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			Rubrics: map[string]config.RubricConfig{
				"refine": {
					Dimensions: []config.DimensionConfig{
						{Name: "Custom Dim", Weight: 7, Description: "custom"},
					},
				},
			},
		},
	}
	rubric := BuildRubric(RubricRefine, cfg)
	if len(rubric.Dimensions) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(rubric.Dimensions))
	}
	if rubric.Dimensions[0].Name != "Custom Dim" {
		t.Errorf("expected 'Custom Dim', got %q", rubric.Dimensions[0].Name)
	}
	if rubric.Total() != 7 {
		t.Errorf("expected total 7, got %d", rubric.Total())
	}
}

func TestDefaultReviewRubricHasPerformance(t *testing.T) {
	dims := DefaultReviewRubric.Dimensions
	if len(dims) != 6 {
		t.Fatalf("expected 6 review dimensions, got %d", len(dims))
	}
	last := dims[len(dims)-1]
	if last.Name != "Performance" {
		t.Errorf("expected last dimension to be 'Performance', got %q", last.Name)
	}
	if DefaultReviewRubric.Total() != 12 {
		t.Errorf("expected review total 12, got %d", DefaultReviewRubric.Total())
	}
}

func TestDefaultRefineRubricLevels(t *testing.T) {
	dims := DefaultRefineRubric.Dimensions
	if len(dims) == 0 {
		t.Fatal("expected non-empty refine dimensions")
	}
	first := dims[0]
	if len(first.Levels) != 6 {
		t.Errorf("expected 6 levels on first dimension, got %d", len(first.Levels))
	}
	if len(first.Levels[0]) < 2 || first.Levels[0][:2] != "5:" {
		t.Errorf("expected first level to start with '5:', got %q", first.Levels[0])
	}
	if len(first.Levels[5]) < 2 || first.Levels[5][:2] != "0:" {
		t.Errorf("expected last level to start with '0:', got %q", first.Levels[5])
	}
}
