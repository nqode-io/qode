package prompt

// Renderer is the minimal interface required by packages that render prompt templates.
type Renderer interface {
	Render(name string, data TemplateData) (string, error)
	ProjectName() string
}
