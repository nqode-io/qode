package scaffold

type workflowDefinition struct {
	Name        string
	Description string
}

var qodeWorkflows = []workflowDefinition{
	{Name: "qode-ticket-fetch", Description: "Fetch a ticket into the current qode context via MCP."},
	{Name: "qode-plan-refine", Description: "Refine requirements with a worker pass plus scoring pass."},
	{Name: "qode-plan-spec", Description: "Generate the technical specification for the active qode context."},
	{Name: "qode-start", Description: "Start the implementation session from the current qode spec."},
	{Name: "qode-check", Description: "Run qode quality gates such as tests and lint checks."},
	{Name: "qode-review-code", Description: "Run the qode code review workflow."},
	{Name: "qode-review-security", Description: "Run the qode security review workflow."},
	{Name: "qode-pr-create", Description: "Create a pull request for the current qode workflow via MCP."},
	{Name: "qode-pr-resolve", Description: "Resolve open pull-request review comments via MCP."},
	{Name: "qode-knowledge-add-context", Description: "Extract durable lessons learned from the current qode context."},
}
