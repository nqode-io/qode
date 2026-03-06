package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	notionAPIBase    = "https://api.notion.com"
	notionAPIVersion = "2022-06-28"
	// notionMaxResponseBytes limits API response body reads to 1 MB.
	notionMaxResponseBytes = 1 << 20
)

// NotionProvider fetches pages from Notion.
type NotionProvider struct {
	apiBase string // empty → notionAPIBase (production default)
}

func (p *NotionProvider) Name() string { return "notion" }

func (p *NotionProvider) CanHandle(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(u.Host, "www.")
	if host == "notion.so" {
		return true
	}
	return host == "notion.site" || strings.HasSuffix(host, ".notion.site")
}

func (p *NotionProvider) Fetch(rawURL string) (*Ticket, error) {
	pageID, err := parseNotionURL(rawURL)
	if err != nil {
		return nil, err
	}

	apiKey := os.Getenv("NOTION_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY environment variable not set\nSet it with: export NOTION_API_KEY=your-token")
	}

	title, err := p.fetchPage(pageID, apiKey)
	if err != nil {
		return nil, err
	}

	blocks, err := p.fetchBlocks(pageID, apiKey)
	if err != nil {
		return nil, err
	}

	body := blocksToMarkdown(blocks)

	return &Ticket{
		ID:    pageID,
		Title: title,
		Body:  body,
		URL:   rawURL,
	}, nil
}

func parseNotionURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("could not parse Notion URL: %w", err)
	}

	// Take the last path segment, strip hyphens, and extract the trailing 32 hex chars.
	cleaned := path.Clean(u.Path)
	segments := strings.Split(cleaned, "/")
	lastSeg := segments[len(segments)-1]
	dehyphenated := strings.ReplaceAll(lastSeg, "-", "")

	// The page ID is always the last 32 characters of the slug (all hex).
	if len(dehyphenated) < 32 {
		return "", fmt.Errorf("could not extract Notion page ID from URL: %s", rawURL)
	}
	candidate := dehyphenated[len(dehyphenated)-32:]
	if !isHex32(candidate) {
		return "", fmt.Errorf("could not extract Notion page ID from URL: %s", rawURL)
	}

	return formatNotionUUID(candidate), nil
}

func isHex32(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// formatNotionUUID formats a 32-char hex string as 8-4-4-4-12 UUID.
func formatNotionUUID(hex string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32])
}

// --- Notion API types ---

type notionPage struct {
	ID         string                    `json:"id"`
	Properties map[string]notionProperty `json:"properties"`
}

type notionProperty struct {
	Type  string          `json:"type"`
	Title []richTextEntry `json:"title,omitempty"`
}

type richTextEntry struct {
	PlainText   string         `json:"plain_text"`
	Href        *string        `json:"href"`
	Annotations *richTextAnnot `json:"annotations,omitempty"`
}

type richTextAnnot struct {
	Bold          bool `json:"bold"`
	Italic        bool `json:"italic"`
	Strikethrough bool `json:"strikethrough"`
	Code          bool `json:"code"`
}

type notionBlockList struct {
	Results    []notionBlock `json:"results"`
	HasMore    bool          `json:"has_more"`
	NextCursor *string       `json:"next_cursor"`
}

type notionBlock struct {
	ID               string        `json:"id"`
	Type             string        `json:"type"`
	HasChildren      bool          `json:"has_children"`
	Paragraph        *blockContent `json:"paragraph,omitempty"`
	Heading1         *blockContent `json:"heading_1,omitempty"`
	Heading2         *blockContent `json:"heading_2,omitempty"`
	Heading3         *blockContent `json:"heading_3,omitempty"`
	BulletedListItem *blockContent `json:"bulleted_list_item,omitempty"`
	NumberedListItem *blockContent `json:"numbered_list_item,omitempty"`
	ToDo             *toDoContent  `json:"to_do,omitempty"`
	Code             *codeContent  `json:"code,omitempty"`
	Quote            *blockContent `json:"quote,omitempty"`
	Toggle           *blockContent `json:"toggle,omitempty"`
	Callout          *blockContent `json:"callout,omitempty"`
	Children         []notionBlock `json:"-"`
}

type blockContent struct {
	RichText []richTextEntry `json:"rich_text"`
}

type toDoContent struct {
	RichText []richTextEntry `json:"rich_text"`
	Checked  bool            `json:"checked"`
}

type codeContent struct {
	RichText []richTextEntry `json:"rich_text"`
	Language string          `json:"language"`
}

// --- API methods ---

func (p *NotionProvider) fetchPage(pageID, apiKey string) (string, error) {
	base := p.apiBase
	if base == "" {
		base = notionAPIBase
	}
	apiURL := fmt.Sprintf("%s/v1/pages/%s", base, pageID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	p.setHeaders(req, apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching Notion page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkNotionResponse(resp); err != nil {
		return "", err
	}

	var page notionPage
	if err := json.NewDecoder(io.LimitReader(resp.Body, notionMaxResponseBytes)).Decode(&page); err != nil {
		return "", fmt.Errorf("decoding Notion page response: %w", err)
	}

	return extractTitle(page), nil
}

func (p *NotionProvider) fetchBlocks(pageID, apiKey string) ([]notionBlock, error) {
	base := p.apiBase
	if base == "" {
		base = notionAPIBase
	}

	blocks, err := p.fetchBlockChildren(base, pageID, apiKey)
	if err != nil {
		return nil, err
	}

	// Fetch one level of nested children.
	for i := range blocks {
		if blocks[i].HasChildren {
			children, err := p.fetchBlockChildren(base, blocks[i].ID, apiKey)
			if err != nil {
				return nil, err
			}
			blocks[i].Children = children
		}
	}

	return blocks, nil
}

func (p *NotionProvider) fetchBlockChildren(base, blockID, apiKey string) ([]notionBlock, error) {
	var allBlocks []notionBlock
	var cursor string

	for {
		params := url.Values{}
		params.Set("page_size", "100")
		if cursor != "" {
			params.Set("start_cursor", cursor)
		}
		apiURL := fmt.Sprintf("%s/v1/blocks/%s/children?%s", base, blockID, params.Encode())

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		p.setHeaders(req, apiKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching Notion blocks: %w", err)
		}

		if err := checkNotionResponse(resp); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}

		var list notionBlockList
		err = json.NewDecoder(io.LimitReader(resp.Body, notionMaxResponseBytes)).Decode(&list)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decoding Notion blocks response: %w", err)
		}

		allBlocks = append(allBlocks, list.Results...)

		if !list.HasMore || list.NextCursor == nil {
			break
		}
		cursor = *list.NextCursor
	}

	return allBlocks, nil
}

func (*NotionProvider) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Accept", "application/json")
}

func checkNotionResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("notion API returned 401 — check NOTION_API_KEY\nSet it with: export NOTION_API_KEY=your-token")
	case http.StatusForbidden:
		return fmt.Errorf("notion API returned 403 — ensure the integration has access to this page\nShare the page with your integration at notion.so/my-integrations")
	case http.StatusNotFound:
		return fmt.Errorf("notion page not found — ensure the page is shared with your integration\nManage integrations at notion.so/my-integrations")
	case http.StatusTooManyRequests:
		return fmt.Errorf("notion API rate limited — try again in a few seconds")
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("notion API returned %d: %s", resp.StatusCode, string(body))
	}
}

// --- Content extraction ---

func extractTitle(page notionPage) string {
	for _, prop := range page.Properties {
		if prop.Type == "title" && len(prop.Title) > 0 {
			return renderPlainText(prop.Title)
		}
	}
	return ""
}

func renderPlainText(entries []richTextEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(e.PlainText)
	}
	return sb.String()
}

func renderRichText(entries []richTextEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		text := e.PlainText
		if a := e.Annotations; a != nil {
			if a.Code {
				text = "`" + text + "`"
			}
			if a.Bold {
				text = "**" + text + "**"
			}
			if a.Italic {
				text = "*" + text + "*"
			}
			if a.Strikethrough {
				text = "~~" + text + "~~"
			}
		}
		if e.Href != nil && *e.Href != "" {
			text = "[" + text + "](" + *e.Href + ")"
		}
		sb.WriteString(text)
	}
	return sb.String()
}

// --- Block-to-Markdown conversion ---

func blocksToMarkdown(blocks []notionBlock) string {
	var sb strings.Builder
	numberedIdx := 0

	for _, b := range blocks {
		line := blockToLine(b, &numberedIdx)
		if line == "" && b.Type != "divider" {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")

		// Render children indented.
		for _, child := range b.Children {
			childIdx := 0
			childLine := blockToLine(child, &childIdx)
			if childLine != "" {
				sb.WriteString("  " + childLine + "\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func blockToLine(b notionBlock, numberedIdx *int) string {
	// Reset numbered list counter when we leave a numbered list.
	if b.Type != "numbered_list_item" {
		*numberedIdx = 0
	}

	switch b.Type {
	case "paragraph":
		return richTextFromContent(b.Paragraph)
	case "heading_1":
		return "# " + richTextFromContent(b.Heading1)
	case "heading_2":
		return "## " + richTextFromContent(b.Heading2)
	case "heading_3":
		return "### " + richTextFromContent(b.Heading3)
	case "bulleted_list_item":
		return "- " + richTextFromContent(b.BulletedListItem)
	case "numbered_list_item":
		*numberedIdx++
		return fmt.Sprintf("%d. %s", *numberedIdx, richTextFromContent(b.NumberedListItem))
	case "to_do":
		if b.ToDo == nil {
			return ""
		}
		check := " "
		if b.ToDo.Checked {
			check = "x"
		}
		return fmt.Sprintf("- [%s] %s", check, renderRichText(b.ToDo.RichText))
	case "code":
		if b.Code == nil {
			return ""
		}
		lang := b.Code.Language
		text := renderRichText(b.Code.RichText)
		return fmt.Sprintf("```%s\n%s\n```", lang, text)
	case "quote":
		return "> " + richTextFromContent(b.Quote)
	case "divider":
		return "---"
	case "toggle":
		return richTextFromContent(b.Toggle)
	case "callout":
		return richTextFromContent(b.Callout)
	default:
		return ""
	}
}

func richTextFromContent(c *blockContent) string {
	if c == nil {
		return ""
	}
	return renderRichText(c.RichText)
}
