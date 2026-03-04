package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	githubAPIBase          = "https://api.github.com"
	githubAPIVersionHeader = "2022-11-28"
)

// GitHubProvider fetches issues from GitHub.
type GitHubProvider struct {
	apiBase string // empty → githubAPIBase (production default)
}

type githubIssue struct {
	Number  int     `json:"number"`
	Title   string  `json:"title"`
	Body    *string `json:"body"`
	HTMLURL string  `json:"html_url"`
}

func (p *GitHubProvider) Name() string { return "github" }

func (p *GitHubProvider) CanHandle(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "github.com" {
		return false
	}
	segments := strings.Split(path.Clean(u.Path), "/")
	return len(segments) >= 5 && segments[3] == "issues" && isNumeric(segments[4])
}

func (p *GitHubProvider) Fetch(rawURL string) (*Ticket, error) {
	owner, repo, number, err := parseGitHubURL(rawURL)
	if err != nil {
		return nil, err
	}
	issue, err := p.fetchIssue(owner, repo, number)
	if err != nil {
		return nil, err
	}
	body := ""
	if issue.Body != nil {
		body = *issue.Body
	}
	return &Ticket{
		ID:    strconv.Itoa(issue.Number),
		Title: issue.Title,
		Body:  body,
		URL:   rawURL,
	}, nil
}

func parseGitHubURL(rawURL string) (owner, repo string, number int, err error) {
	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", 0, fmt.Errorf("could not parse GitHub URL: %w", parseErr)
	}
	segments := strings.Split(path.Clean(u.Path), "/")
	// Expected: ["", "owner", "repo", "issues", "number"]
	if len(segments) < 5 || segments[3] != "issues" {
		return "", "", 0, fmt.Errorf("could not parse GitHub URL: expected /owner/repo/issues/number")
	}
	n, atoiErr := strconv.Atoi(segments[4])
	if atoiErr != nil {
		return "", "", 0, fmt.Errorf("invalid GitHub issue number: %s", segments[4])
	}
	return url.PathEscape(segments[1]), url.PathEscape(segments[2]), n, nil
}

func (p *GitHubProvider) fetchIssue(owner, repo string, number int) (*githubIssue, error) {
	base := p.apiBase
	if base == "" {
		base = githubAPIBase
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d", base, owner, repo, number)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersionHeader)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub issue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub API returned %d — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub issue not found — if this is a private repository, set GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
	var issue githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decoding GitHub response: %w", err)
	}
	return &issue, nil
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
