package ticket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubProvider_Name(t *testing.T) {
	p := &GitHubProvider{}
	if p.Name() != "github" {
		t.Errorf("expected %q, got %q", "github", p.Name())
	}
}

func TestGitHubProvider_CanHandle(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://github.com/owner/repo/issues/42", true},
		{"https://github.com/owner/repo/issues/42/", true},
		{"https://github.com/owner/repo/pull/42", false},
		{"https://linear.app/team/ENG-123", false},
		{"https://github.com/owner/repo", false},
		{"https://github.com/owner/repo/issues/abc", false},
		{"https://github.com/owner/repo/issues/", false},
	}
	p := &GitHubProvider{}
	for _, tc := range cases {
		got := p.CanHandle(tc.url)
		if got != tc.want {
			t.Errorf("CanHandle(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestParseGitHubURL_Valid(t *testing.T) {
	owner, repo, number, err := parseGitHubURL("https://github.com/my-org/my-repo/issues/99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "my-org" {
		t.Errorf("owner = %q, want %q", owner, "my-org")
	}
	if repo != "my-repo" {
		t.Errorf("repo = %q, want %q", repo, "my-repo")
	}
	if number != 99 {
		t.Errorf("number = %d, want %d", number, 99)
	}
}

func TestParseGitHubURL_InvalidPath(t *testing.T) {
	cases := []string{
		"https://github.com/owner/repo",
		"https://github.com/owner/repo/pull/42",
		"https://github.com/owner",
	}
	for _, rawURL := range cases {
		_, _, _, err := parseGitHubURL(rawURL)
		if err == nil {
			t.Errorf("parseGitHubURL(%q): expected error, got nil", rawURL)
		}
	}
}

func TestParseGitHubURL_NonNumericNumber(t *testing.T) {
	_, _, _, err := parseGitHubURL("https://github.com/owner/repo/issues/abc")
	if err == nil {
		t.Fatal("expected error for non-numeric issue number, got nil")
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Errorf("error %q should mention the invalid segment", err.Error())
	}
}

func TestGitHubProvider_Fetch_Success(t *testing.T) {
	body := "This is the issue body."
	issue := githubIssue{
		Number:  42,
		Title:   "Fix the bug",
		Body:    &body,
		HTMLURL: "https://github.com/owner/repo/issues/42",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	rawURL := "https://github.com/owner/repo/issues/42"
	ticket, err := p.Fetch(rawURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != "42" {
		t.Errorf("ID = %q, want %q", ticket.ID, "42")
	}
	if ticket.Title != "Fix the bug" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Fix the bug")
	}
	if ticket.Body != body {
		t.Errorf("Body = %q, want %q", ticket.Body, body)
	}
	if ticket.URL != rawURL {
		t.Errorf("URL = %q, want %q", ticket.URL, rawURL)
	}
}

func TestGitHubProvider_Fetch_NullBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"number":1,"title":"No body","body":null}`)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Body != "" {
		t.Errorf("Body = %q, want empty string for null body", ticket.Body)
	}
}

func TestGitHubProvider_Fetch_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error %q should reference GITHUB_TOKEN", err.Error())
	}
}

func TestGitHubProvider_Fetch_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q should contain '401'", err.Error())
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error %q should reference GITHUB_TOKEN", err.Error())
	}
}

func TestGitHubProvider_Fetch_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error %q should contain '403'", err.Error())
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error %q should reference GITHUB_TOKEN", err.Error())
	}
}

func TestGitHubProvider_Fetch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should contain '500'", err.Error())
	}
}

func TestGitHubProvider_Fetch_SendsAuthHeader(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"number":1,"title":"T","body":null}`)
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, _ = p.Fetch("https://github.com/owner/repo/issues/1")
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
}

func TestGitHubProvider_Fetch_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "not json")
	}))
	defer server.Close()

	p := &GitHubProvider{apiBase: server.URL}
	_, err := p.Fetch("https://github.com/owner/repo/issues/1")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "decoding GitHub response") {
		t.Errorf("error %q should contain 'decoding GitHub response'", err.Error())
	}
}
