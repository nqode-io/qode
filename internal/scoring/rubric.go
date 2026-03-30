package scoring

import "github.com/nqode/qode/internal/config"

// RubricKind identifies which rubric to use.
type RubricKind string

const (
	RubricRefine   RubricKind = "refine"
	RubricReview   RubricKind = "review"
	RubricSecurity RubricKind = "security"
)

// Rubric describes a scoring scheme.
type Rubric struct {
	Kind       RubricKind
	Dimensions []Dimension
}

// Total returns the sum of all dimension weights.
func (r Rubric) Total() int {
	total := 0
	for _, d := range r.Dimensions {
		total += d.Weight
	}
	return total
}

// Dimension is one scoring axis in a rubric.
type Dimension struct {
	Name   string
	Weight int      // points available
	Desc   string
	Levels []string // full labelled strings highest-first, e.g. "5: Perfect restatement..."
}

// DefaultRefineRubric is the 5 × 5 requirements-refinement rubric.
var DefaultRefineRubric = Rubric{
	Kind: RubricRefine,
	Dimensions: []Dimension{
		{
			Name: "Problem Understanding", Weight: 5,
			Desc: "Correct restatement, all ambiguities resolved, user need clear",
			Levels: []string{
				"5: Perfect restatement, all ambiguities resolved, user need crystal clear",
				"4: Good understanding, minor gaps",
				"3: Adequate but surface-level",
				"2: Partial understanding, significant gaps",
				"1: Mostly incorrect or too vague",
				"0: Missing or completely wrong",
			},
		},
		{
			Name: "Technical Analysis", Weight: 5,
			Desc: "All affected components identified, specific file/API references",
			Levels: []string{
				"5: Identifies all affected components, correct patterns, specific file/API references",
				"4: Mostly correct, minor omissions",
				"3: Covers main points, misses some details",
				"2: Shallow analysis, important components missed",
				"1: Mostly incorrect or generic",
				"0: Missing",
			},
		},
		{
			Name: "Risk & Edge Cases", Weight: 5,
			Desc: "Comprehensive risks, specific edge cases with mitigation",
			Levels: []string{
				"5: Comprehensive risk analysis, specific edge cases with mitigation",
				"4: Good coverage, minor gaps",
				"3: Some risks identified, misses important ones",
				"2: Superficial",
				"1: Generic or incorrect",
				"0: Missing",
			},
		},
		{
			Name: "Completeness", Weight: 5,
			Desc: "All acceptance criteria, implicit requirements, scope clear",
			Levels: []string{
				"5: All acceptance criteria captured, implicit requirements identified, scope clear",
				"4: Nearly complete, minor omissions",
				"3: Main requirements captured, gaps exist",
				"2: Significant gaps",
				"1: Incomplete",
				"0: Missing",
			},
		},
		{
			Name: "Actionability", Weight: 5,
			Desc: "Concrete tasks, correct order, prerequisites, each one commit",
			Levels: []string{
				"5: Clear concrete tasks, correct order, prerequisites identified, each task is one commit",
				"4: Mostly actionable, minor improvements possible",
				"3: Tasks defined but too coarse or unclear",
				"2: Hard to act on",
				"1: Very vague",
				"0: Missing",
			},
		},
	},
}

// DefaultReviewRubric is the 12-point code review rubric (6 dimensions × weight 2).
var DefaultReviewRubric = Rubric{
	Kind: RubricReview,
	Dimensions: []Dimension{
		{Name: "Correctness", Weight: 2, Desc: "Implements spec correctly, no logic bugs"},
		{Name: "Code Quality", Weight: 2, Desc: "Readable, maintainable, well-named"},
		{Name: "Architecture", Weight: 2, Desc: "Follows patterns, correct separation of concerns"},
		{Name: "Error Handling", Weight: 2, Desc: "All error paths handled explicitly"},
		{Name: "Testing", Weight: 2, Desc: "Tests present and cover edge cases"},
		{Name: "Performance", Weight: 2, Desc: "No obvious performance issues, unnecessary allocations, or blocking calls"},
	},
}

// DefaultSecurityRubric is the 10-point security review rubric.
var DefaultSecurityRubric = Rubric{
	Kind: RubricSecurity,
	Dimensions: []Dimension{
		{Name: "Injection Prevention", Weight: 2, Desc: "No SQL/command/template injection vectors"},
		{Name: "Auth & AuthZ", Weight: 2, Desc: "Authentication bypass and IDOR prevention"},
		{Name: "Data Exposure", Weight: 2, Desc: "No PII leak, secure storage"},
		{Name: "Input Validation", Weight: 2, Desc: "All inputs validated and sanitised"},
		{Name: "Dependency Safety", Weight: 2, Desc: "No known CVEs in new deps"},
	},
}

// BuildRubric returns the rubric for kind, using cfg overrides when present.
// If cfg is nil or no rubric is configured for kind, the default rubric is returned.
// A configured rubric fully replaces the default — no partial merge.
func BuildRubric(kind RubricKind, cfg *config.Config) Rubric {
	if cfg != nil {
		if rc, ok := cfg.Scoring.Rubrics[string(kind)]; ok && len(rc.Dimensions) > 0 {
			dims := make([]Dimension, len(rc.Dimensions))
			for i, dc := range rc.Dimensions {
				dims[i] = Dimension{
					Name:   dc.Name,
					Weight: dc.Weight,
					Desc:   dc.Description,
					Levels: dc.Levels,
				}
			}
			return Rubric{Kind: kind, Dimensions: dims}
		}
	}
	switch kind {
	case RubricReview:
		return DefaultReviewRubric
	case RubricSecurity:
		return DefaultSecurityRubric
	default:
		return DefaultRefineRubric
	}
}
