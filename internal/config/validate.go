package config

import (
	"errors"
	"fmt"
	"strings"
)

// validRubricKeys lists the only rubric keys recognised by the scoring engine.
var validRubricKeys = map[string]struct{}{
	"refine":   {},
	"review":   {},
	"security": {},
}

// Validate checks that the loaded Config contains only sensible values.
// All violations are collected and returned as a single joined error.
func (c *Config) Validate() error {
	var errs []string

	if c.Review.MinCodeScore < 0 {
		errs = append(errs, fmt.Sprintf("review.min_code_score must be non-negative, got %.2f", c.Review.MinCodeScore))
	}
	if c.Review.MinSecurityScore < 0 {
		errs = append(errs, fmt.Sprintf("review.min_security_score must be non-negative, got %.2f", c.Review.MinSecurityScore))
	}
	if c.Scoring.TargetScore < 0 {
		errs = append(errs, fmt.Sprintf("scoring.target_score must be non-negative, got %d", c.Scoring.TargetScore))
	}

	for key, rubric := range c.Scoring.Rubrics {
		if _, ok := validRubricKeys[key]; !ok {
			errs = append(errs, fmt.Sprintf("scoring.rubrics: unknown rubric key %q (allowed: refine, review, security)", key))
		}

		if len(rubric.Dimensions) == 0 {
			errs = append(errs, fmt.Sprintf("scoring.rubrics[%q]: dimensions must not be empty", key))
			continue
		}

		for i, dim := range rubric.Dimensions {
			if dim.Name == "" {
				errs = append(errs, fmt.Sprintf("scoring.rubrics[%q].dimensions[%d]: name must not be empty", key, i))
			}
			if dim.Weight <= 0 {
				errs = append(errs, fmt.Sprintf("scoring.rubrics[%q].dimensions[%d] (%q): weight must be positive, got %d", key, i, dim.Name, dim.Weight))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
