package scoring

import "github.com/nqode/qode/internal/config"

// RubricKind identifies which rubric to use.
type RubricKind string

// Supported rubric kinds used to select scoring schemes.
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

// configToDimensions converts a slice of config.DimensionConfig to []Dimension.
func configToDimensions(dcs []config.DimensionConfig) []Dimension {
	dims := make([]Dimension, len(dcs))
	for i, dc := range dcs {
		dims[i] = Dimension{
			Name:   dc.Name,
			Weight: dc.Weight,
			Desc:   dc.Description,
			Levels: dc.Levels,
		}
	}
	return dims
}

// defaultRubrics is the single source of truth for built-in rubric data,
// derived from config.DefaultRubricConfigs() at package init time.
var defaultRubrics = config.DefaultRubricConfigs()

// DefaultRefineRubric is the 5 × 5 requirements-refinement rubric.
var DefaultRefineRubric = Rubric{
	Kind:       RubricRefine,
	Dimensions: configToDimensions(defaultRubrics[string(RubricRefine)].Dimensions),
}

// DefaultReviewRubric is the 12-point code review rubric (6 dimensions × weight 2).
var DefaultReviewRubric = Rubric{
	Kind:       RubricReview,
	Dimensions: configToDimensions(defaultRubrics[string(RubricReview)].Dimensions),
}

// DefaultSecurityRubric is the 10-point security review rubric.
var DefaultSecurityRubric = Rubric{
	Kind:       RubricSecurity,
	Dimensions: configToDimensions(defaultRubrics[string(RubricSecurity)].Dimensions),
}

// BuildRubric returns the rubric for kind, using cfg overrides when present.
// If cfg is nil or no rubric is configured for kind, the default rubric is returned.
// A configured rubric fully replaces the default — no partial merge.
func BuildRubric(kind RubricKind, cfg *config.Config) Rubric {
	if cfg != nil {
		if rc, ok := cfg.Scoring.Rubrics[string(kind)]; ok && len(rc.Dimensions) > 0 {
			return Rubric{Kind: kind, Dimensions: configToDimensions(rc.Dimensions)}
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
