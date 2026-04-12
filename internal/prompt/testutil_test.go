package prompt

// minimalTemplateData returns a minimal TemplateDataBuilder suitable for tests
// that don't depend on specific project values.
func minimalTemplateData() *TemplateDataBuilder {
	return NewTemplateData("test-project")
}
