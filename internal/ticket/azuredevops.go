package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// AzureDevOpsProvider fetches work items from Azure DevOps.
type AzureDevOpsProvider struct{}

func (p *AzureDevOpsProvider) Name() string { return "azure-devops" }

func (p *AzureDevOpsProvider) CanHandle(rawURL string) bool {
	return hostContains(rawURL, "dev.azure.com") || hostContains(rawURL, "visualstudio.com")
}

func (p *AzureDevOpsProvider) Fetch(rawURL string) (*Ticket, error) {
	org, project, id, err := extractAzureParams(rawURL)
	if err != nil {
		return nil, err
	}

	pat := os.Getenv("AZURE_DEVOPS_PAT")
	if pat == "" {
		return nil, fmt.Errorf("AZURE_DEVOPS_PAT environment variable not set\nSet it with: export AZURE_DEVOPS_PAT=your-token")
	}

	apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/wit/workitems/%s?api-version=7.0",
		url.PathEscape(org), url.PathEscape(project), url.PathEscape(id))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", pat)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Azure DevOps work item: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("azure DevOps API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     int    `json:"id"`
		Fields struct {
			Title       string `json:"System.Title"`
			Description string `json:"System.Description"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding Azure DevOps response: %w", err)
	}

	return &Ticket{
		ID:    fmt.Sprintf("%d", result.ID),
		Title: result.Fields.Title,
		Body:  stripHTML(result.Fields.Description),
		URL:   rawURL,
	}, nil
}

var azureWorkItemRe = regexp.MustCompile(`_workitems/edit/(\d+)`)

func extractAzureParams(rawURL string) (org, project, id string, err error) {
	// https://dev.azure.com/{org}/{project}/_workitems/edit/{id}
	m := azureWorkItemRe.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return "", "", "", fmt.Errorf("could not extract work item ID from URL: %s", rawURL)
	}
	id = m[1]

	// Extract org and project from path.
	trimmed := strings.TrimPrefix(rawURL, "https://dev.azure.com/")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("could not extract org/project from URL: %s", rawURL)
	}
	return parts[0], parts[1], id, nil
}

var (
	dangerTagRe = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed)[^>]*>.*?</(script|style|iframe|object|embed)>`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
)

// stripHTML removes HTML tags and the content of dangerous tags from a string.
func stripHTML(s string) string {
	s = dangerTagRe.ReplaceAllString(s, "")
	return htmlTagRe.ReplaceAllString(s, "")
}
