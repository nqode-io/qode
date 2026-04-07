package config

import (
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		Review: ReviewConfig{
			MinCodeScore:     10.0,
			MinSecurityScore: 8.0,
		},
		Scoring: ScoringConfig{
			TargetScore: 0,
			Rubrics: map[string]RubricConfig{
				"refine": {
					Dimensions: []DimensionConfig{
						{Name: "Clarity", Weight: 5},
					},
				},
				"review": {
					Dimensions: []DimensionConfig{
						{Name: "Correctness", Weight: 5},
					},
				},
				"security": {
					Dimensions: []DimensionConfig{
						{Name: "AuthN", Weight: 5},
					},
				},
			},
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestValidate_NegativeMinCodeScore(t *testing.T) {
	cfg := validConfig()
	cfg.Review.MinCodeScore = -1.0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative MinCodeScore")
	}
	if !strings.Contains(err.Error(), "min_code_score") {
		t.Errorf("error should mention min_code_score, got: %v", err)
	}
}

func TestValidate_NegativeMinSecurityScore(t *testing.T) {
	cfg := validConfig()
	cfg.Review.MinSecurityScore = -5.0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative MinSecurityScore")
	}
	if !strings.Contains(err.Error(), "min_security_score") {
		t.Errorf("error should mention min_security_score, got: %v", err)
	}
}

func TestValidate_NegativeTargetScore(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.TargetScore = -3
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative TargetScore")
	}
	if !strings.Contains(err.Error(), "target_score") {
		t.Errorf("error should mention target_score, got: %v", err)
	}
}

func TestValidate_UnknownRubricKey(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.Rubrics["unknown"] = RubricConfig{
		Dimensions: []DimensionConfig{{Name: "Foo", Weight: 1}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown rubric key")
	}
	if !strings.Contains(err.Error(), "unknown rubric key") {
		t.Errorf("error should mention unknown rubric key, got: %v", err)
	}
}

func TestValidate_ZeroWeightDimension(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.Rubrics["refine"] = RubricConfig{
		Dimensions: []DimensionConfig{
			{Name: "Clarity", Weight: 0},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for zero-weight dimension")
	}
	if !strings.Contains(err.Error(), "weight must be positive") {
		t.Errorf("error should mention weight must be positive, got: %v", err)
	}
}

func TestValidate_NegativeWeightDimension(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.Rubrics["review"] = RubricConfig{
		Dimensions: []DimensionConfig{
			{Name: "Correctness", Weight: -2},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative-weight dimension")
	}
	if !strings.Contains(err.Error(), "weight must be positive") {
		t.Errorf("error should mention weight must be positive, got: %v", err)
	}
}

func TestValidate_EmptyDimensionName(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.Rubrics["security"] = RubricConfig{
		Dimensions: []DimensionConfig{
			{Name: "", Weight: 5},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty dimension name")
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("error should mention name must not be empty, got: %v", err)
	}
}

func TestValidate_EmptyDimensionsSlice(t *testing.T) {
	cfg := validConfig()
	cfg.Scoring.Rubrics["refine"] = RubricConfig{
		Dimensions: []DimensionConfig{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty dimensions slice")
	}
	if !strings.Contains(err.Error(), "dimensions must not be empty") {
		t.Errorf("error should mention dimensions must not be empty, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.Review.MinCodeScore = -1.0
	cfg.Review.MinSecurityScore = -2.0
	cfg.Scoring.TargetScore = -5

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for multiple invalid fields")
	}

	msg := err.Error()
	if !strings.Contains(msg, "min_code_score") {
		t.Errorf("error should mention min_code_score, got: %v", msg)
	}
	if !strings.Contains(msg, "min_security_score") {
		t.Errorf("error should mention min_security_score, got: %v", msg)
	}
	if !strings.Contains(msg, "target_score") {
		t.Errorf("error should mention target_score, got: %v", msg)
	}
}
