package ticket

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nqode/qode/internal/config"
)

// Ticket is the retrieved ticket data.
type Ticket struct {
	ID    string
	Title string
	Body  string
	URL   string
}

// Provider fetches a ticket from an external system.
type Provider interface {
	Name() string
	CanHandle(rawURL string) bool
	Fetch(rawURL string) (*Ticket, error)
}

var providers []Provider

func init() {
	providers = []Provider{
		&JiraProvider{},
		&AzureDevOpsProvider{},
		&LinearProvider{},
	}
}

// DetectProvider finds the right provider for a URL and configures it.
func DetectProvider(rawURL string, cfg config.TicketSystemConfig) (Provider, error) {
	for _, p := range providers {
		if p.CanHandle(rawURL) {
			return p, nil
		}
	}

	// Configured system type takes precedence.
	if cfg.Type != "" && cfg.Type != "manual" {
		return nil, fmt.Errorf("ticket URL %q not handled by configured system %q — check qode.yaml", rawURL, cfg.Type)
	}

	return nil, fmt.Errorf("no ticket provider found for URL %q — add context/ticket.md manually", rawURL)
}

// hostContains is a helper for URL matching.
func hostContains(rawURL, sub string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, sub)
	}
	return strings.Contains(u.Host, sub)
}
