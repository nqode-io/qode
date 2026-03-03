package scoring

// RubricKind identifies which rubric to use.
type RubricKind string

const (
	RubricRefine   RubricKind = "refine"   // 5 × 5 = 25 points
	RubricReview   RubricKind = "review"   // 10-point scale
	RubricSecurity RubricKind = "security" // 10-point scale
)

// Rubric describes a scoring scheme.
type Rubric struct {
	Kind       RubricKind
	MaxScore   int
	Dimensions []Dimension
}

// Dimension is one scoring axis in a rubric.
type Dimension struct {
	Name   string
	Weight int // points available
	Desc   string
}

// RefineRubric is the 5 × 5 requirements-refinement rubric.
var RefineRubric = Rubric{
	Kind:     RubricRefine,
	MaxScore: 25,
	Dimensions: []Dimension{
		{Name: "Problem Understanding", Weight: 5, Desc: "Correct restatement, all ambiguities resolved, user need clear"},
		{Name: "Technical Analysis", Weight: 5, Desc: "All affected components identified, specific file/API references"},
		{Name: "Risk & Edge Cases", Weight: 5, Desc: "Comprehensive risks, specific edge cases with mitigation"},
		{Name: "Completeness", Weight: 5, Desc: "All acceptance criteria, implicit requirements, scope clear"},
		{Name: "Actionability", Weight: 5, Desc: "Concrete tasks, correct order, prerequisites, each one commit"},
	},
}

// ReviewRubric is the 10-point code review rubric.
var ReviewRubric = Rubric{
	Kind:     RubricReview,
	MaxScore: 10,
	Dimensions: []Dimension{
		{Name: "Correctness", Weight: 2, Desc: "Implements spec correctly, no logic bugs"},
		{Name: "Code Quality", Weight: 2, Desc: "Readable, maintainable, well-named"},
		{Name: "Architecture", Weight: 2, Desc: "Follows patterns, correct separation of concerns"},
		{Name: "Error Handling", Weight: 2, Desc: "All error paths handled explicitly"},
		{Name: "Testing", Weight: 2, Desc: "Tests present and cover edge cases"},
	},
}

// SecurityRubric is the 10-point security review rubric.
var SecurityRubric = Rubric{
	Kind:     RubricSecurity,
	MaxScore: 10,
	Dimensions: []Dimension{
		{Name: "Injection Prevention", Weight: 2, Desc: "No SQL/command/template injection vectors"},
		{Name: "Auth & AuthZ", Weight: 2, Desc: "Authentication bypass and IDOR prevention"},
		{Name: "Data Exposure", Weight: 2, Desc: "No PII leak, secure storage"},
		{Name: "Input Validation", Weight: 2, Desc: "All inputs validated and sanitised"},
		{Name: "Dependency Safety", Weight: 2, Desc: "No known CVEs in new deps"},
	},
}
