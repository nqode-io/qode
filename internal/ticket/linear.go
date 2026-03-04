package ticket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
)

// LinearProvider fetches issues from Linear.
type LinearProvider struct{}

func (p *LinearProvider) Name() string { return "linear" }

func (p *LinearProvider) CanHandle(rawURL string) bool {
	return hostContains(rawURL, "linear.app")
}

func (p *LinearProvider) Fetch(rawURL string) (*Ticket, error) {
	issueID := extractLinearID(rawURL)
	if issueID == "" {
		return nil, fmt.Errorf("could not extract Linear issue ID from URL: %s", rawURL)
	}

	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LINEAR_API_KEY environment variable not set\nSet it with: export LINEAR_API_KEY=your-key")
	}

	query := fmt.Sprintf(`{"query": "{ issue(id: \"%s\") { identifier title description } }"}`, issueID)
	req, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBufferString(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Linear issue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("linear API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Issue struct {
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"issue"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding Linear response: %w", err)
	}

	issue := result.Data.Issue
	return &Ticket{
		ID:    issue.Identifier,
		Title: issue.Title,
		Body:  issue.Description,
		URL:   rawURL,
	}, nil
}

var linearIDRe = regexp.MustCompile(`[A-Z]+-\d+`)

func extractLinearID(rawURL string) string {
	return linearIDRe.FindString(rawURL)
}
