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

// JiraProvider fetches tickets from Jira Cloud or Server.
type JiraProvider struct{}

func (p *JiraProvider) Name() string { return "jira" }

func (p *JiraProvider) CanHandle(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return isAllowedJiraHost(u.Host)
}

func isAllowedJiraHost(host string) bool {
	return host == "atlassian.net" ||
		strings.HasSuffix(host, ".atlassian.net") ||
		strings.HasPrefix(host, "jira.")
}

func (p *JiraProvider) Fetch(rawURL string) (*Ticket, error) {
	issueKey := extractJiraKey(rawURL)
	if issueKey == "" {
		return nil, fmt.Errorf("could not extract Jira issue key from URL: %s", rawURL)
	}

	baseURL, err := extractJiraBase(rawURL)
	if err != nil {
		return nil, err
	}
	token := os.Getenv("JIRA_API_TOKEN")
	email := os.Getenv("JIRA_EMAIL")

	if token == "" {
		return nil, fmt.Errorf("JIRA_API_TOKEN environment variable not set\nSet it with: export JIRA_API_TOKEN=your-token")
	}

	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", baseURL, issueKey)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Jira issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description struct {
				Content []struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"content"`
			} `json:"description"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding Jira response: %w", err)
	}

	// Extract plain text from Atlassian Document Format.
	var body strings.Builder
	for _, block := range result.Fields.Description.Content {
		for _, inline := range block.Content {
			body.WriteString(inline.Text)
		}
		body.WriteString("\n")
	}

	return &Ticket{
		ID:    result.Key,
		Title: result.Fields.Summary,
		Body:  body.String(),
		URL:   rawURL,
	}, nil
}

var jiraKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func extractJiraKey(rawURL string) string {
	return jiraKeyRe.FindString(rawURL)
}

func extractJiraBase(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return "", fmt.Errorf("invalid ticket URL: must use HTTPS")
	}
	if !isAllowedJiraHost(u.Host) {
		return "", fmt.Errorf("URL host %q is not an allowed Jira host", u.Host)
	}
	return u.Scheme + "://" + u.Host, nil
}
