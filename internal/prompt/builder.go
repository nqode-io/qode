package prompt

import "github.com/nqode/qode/internal/scoring"

// TemplateDataBuilder constructs TemplateData with method chaining.
type TemplateDataBuilder struct {
	data TemplateData
}

// NewTemplateData starts building a TemplateData with the project name and branch.
func NewTemplateData(projectName, branch string) *TemplateDataBuilder {
	return &TemplateDataBuilder{
		data: TemplateData{
			Project: TemplateProject{Name: projectName},
			Branch:  branch,
		},
	}
}

// WithIDE sets the target IDE identifier.
func (b *TemplateDataBuilder) WithIDE(ide string) *TemplateDataBuilder {
	b.data.IDE = ide
	return b
}

// WithOutputPath sets the file-write instruction path.
func (b *TemplateDataBuilder) WithOutputPath(p string) *TemplateDataBuilder {
	b.data.OutputPath = p
	return b
}

// WithBranchDir sets the absolute path to the branch state directory.
func (b *TemplateDataBuilder) WithBranchDir(d string) *TemplateDataBuilder {
	b.data.BranchDir = d
	return b
}

// WithRubric sets the scoring rubric.
func (b *TemplateDataBuilder) WithRubric(r scoring.Rubric) *TemplateDataBuilder {
	b.data.Rubric = r
	return b
}

// WithTargetScore sets the pass threshold for refine judge.
func (b *TemplateDataBuilder) WithTargetScore(s int) *TemplateDataBuilder {
	b.data.TargetScore = s
	return b
}

// WithMinPassScore sets the minimum score for code/security reviews.
func (b *TemplateDataBuilder) WithMinPassScore(s float64) *TemplateDataBuilder {
	b.data.MinPassScore = s
	return b
}

// WithTicket sets the inline ticket content.
func (b *TemplateDataBuilder) WithTicket(t string) *TemplateDataBuilder {
	b.data.Ticket = t
	return b
}

// WithAnalysis sets the inline analysis content.
func (b *TemplateDataBuilder) WithAnalysis(a string) *TemplateDataBuilder {
	b.data.Analysis = a
	return b
}

// WithSpec sets the inline spec content.
func (b *TemplateDataBuilder) WithSpec(s string) *TemplateDataBuilder {
	b.data.Spec = s
	return b
}

// WithDiff sets the inline diff content.
func (b *TemplateDataBuilder) WithDiff(d string) *TemplateDataBuilder {
	b.data.Diff = d
	return b
}

// WithExtra sets the inline extra context content.
func (b *TemplateDataBuilder) WithExtra(e string) *TemplateDataBuilder {
	b.data.Extra = e
	return b
}

// WithKB sets the knowledge base file references.
func (b *TemplateDataBuilder) WithKB(kb string) *TemplateDataBuilder {
	b.data.KB = kb
	return b
}

// WithLessons sets the existing lesson summaries.
func (b *TemplateDataBuilder) WithLessons(l string) *TemplateDataBuilder {
	b.data.Lessons = l
	return b
}

// Build returns the constructed TemplateData.
func (b *TemplateDataBuilder) Build() TemplateData {
	return b.data
}
