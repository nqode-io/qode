package ticket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotionProvider_Name(t *testing.T) {
	p := &NotionProvider{}
	if p.Name() != "notion" {
		t.Errorf("expected %q, got %q", "notion", p.Name())
	}
}

func TestNotionProvider_CanHandle(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.notion.so/workspace/My-Page-abc123def456abc123def456abc123de", true},
		{"https://notion.so/abc123def456abc123def456abc123de", true},
		{"https://myteam.notion.site/Page-abc123def456abc123def456abc123de", true},
		{"https://www.notion.so/workspace/Page-abc123", true},
		{"https://notion.site/Page-abc123", true},
		{"https://github.com/owner/repo/issues/42", false},
		{"https://linear.app/team/ENG-123", false},
		{"https://company.atlassian.net/browse/ENG-123", false},
		{"not-a-url", false},
		{"https://fake-notion.so/page", false},
		{"https://notion.so.evil.com/page", false},
	}
	p := &NotionProvider{}
	for _, tc := range cases {
		got := p.CanHandle(tc.url)
		if got != tc.want {
			t.Errorf("CanHandle(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestParseNotionURL_Valid(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "workspace path with title",
			url:  "https://www.notion.so/workspace/My-Page-abc123de1234567890abcdef12345678",
			want: "abc123de-1234-5678-90ab-cdef12345678",
		},
		{
			name: "bare ID",
			url:  "https://notion.so/abc123de1234567890abcdef12345678",
			want: "abc123de-1234-5678-90ab-cdef12345678",
		},
		{
			name: "notion.site with workspace",
			url:  "https://myteam.notion.site/Page-Title-abc123de1234567890abcdef12345678",
			want: "abc123de-1234-5678-90ab-cdef12345678",
		},
		{
			name: "ID with hyphens in URL",
			url:  "https://notion.so/abc123de-1234-5678-90ab-cdef12345678",
			want: "abc123de-1234-5678-90ab-cdef12345678",
		},
		{
			name: "URL with query params",
			url:  "https://notion.so/Page-abc123de1234567890abcdef12345678?v=someid",
			want: "abc123de-1234-5678-90ab-cdef12345678",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseNotionURL(tc.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("parseNotionURL(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}

func TestParseNotionURL_Invalid(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"empty path", "https://notion.so/"},
		{"too short ID", "https://notion.so/abc123"},
		{"non-hex chars", "https://notion.so/Page-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{"not a URL", "not-a-url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseNotionURL(tc.url)
			if err == nil {
				t.Errorf("parseNotionURL(%q): expected error, got nil", tc.url)
			}
		})
	}
}

func TestNotionProvider_Fetch_Success(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")

	pageResp := notionPage{
		ID: "abc123de-1234-5678-90ab-cdef12345678",
		Properties: map[string]notionProperty{
			"Name": {
				Type: "title",
				Title: []richTextEntry{
					{PlainText: "Test Ticket"},
				},
			},
		},
	}

	blocksResp := notionBlockList{
		Results: []notionBlock{
			{
				Type:      "paragraph",
				Paragraph: &blockContent{RichText: []richTextEntry{{PlainText: "Hello world"}}},
			},
			{
				Type:     "heading_1",
				Heading1: &blockContent{RichText: []richTextEntry{{PlainText: "Section"}}},
			},
		},
		HasMore: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/") {
			_ = json.NewEncoder(w).Encode(blocksResp)
		} else {
			_ = json.NewEncoder(w).Encode(pageResp)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://notion.so/Test-abc123de1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Title != "Test Ticket" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Test Ticket")
	}
	if !strings.Contains(ticket.Body, "Hello world") {
		t.Errorf("Body should contain 'Hello world', got %q", ticket.Body)
	}
	if !strings.Contains(ticket.Body, "# Section") {
		t.Errorf("Body should contain '# Section', got %q", ticket.Body)
	}
}

func TestNotionProvider_Fetch_EmptyBody(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/") {
			_, _ = fmt.Fprint(w, `{"results":[],"has_more":false}`)
		} else {
			_, _ = fmt.Fprint(w, `{"id":"abc","properties":{"Name":{"type":"title","title":[{"plain_text":"Empty"}]}}}`)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Body != "" {
		t.Errorf("Body = %q, want empty string", ticket.Body)
	}
}

func TestNotionProvider_Fetch_MissingToken(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "")
	p := &NotionProvider{}
	_, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
	if !strings.Contains(err.Error(), "NOTION_API_KEY") {
		t.Errorf("error %q should reference NOTION_API_KEY", err.Error())
	}
}

func TestNotionProvider_Fetch_Unauthorized(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "bad-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	_, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q should contain '401'", err.Error())
	}
	if !strings.Contains(err.Error(), "NOTION_API_KEY") {
		t.Errorf("error %q should reference NOTION_API_KEY", err.Error())
	}
}

func TestNotionProvider_Fetch_Forbidden(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	_, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error %q should contain '403'", err.Error())
	}
}

func TestNotionProvider_Fetch_NotFound(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	_, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestNotionProvider_Fetch_ServerError(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	_, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should contain '500'", err.Error())
	}
}

func TestNotionProvider_Fetch_SendsAuthHeader(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/") {
			_, _ = fmt.Fprint(w, `{"results":[],"has_more":false}`)
		} else {
			_, _ = fmt.Fprint(w, `{"id":"abc","properties":{"Name":{"type":"title","title":[{"plain_text":"T"}]}}}`)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	_, _ = p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
}

func TestNotionProvider_Fetch_Pagination(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")

	cursor := "next-page"
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/") {
			callCount++
			if r.URL.Query().Get("start_cursor") == "" {
				// First page.
				resp := map[string]interface{}{
					"results": []map[string]interface{}{
						{"type": "paragraph", "paragraph": map[string]interface{}{
							"rich_text": []map[string]string{{"plain_text": "Page 1"}},
						}},
					},
					"has_more":    true,
					"next_cursor": cursor,
				}
				_ = json.NewEncoder(w).Encode(resp)
			} else {
				// Second page.
				resp := map[string]interface{}{
					"results": []map[string]interface{}{
						{"type": "paragraph", "paragraph": map[string]interface{}{
							"rich_text": []map[string]string{{"plain_text": "Page 2"}},
						}},
					},
					"has_more": false,
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		} else {
			_, _ = fmt.Fprint(w, `{"id":"abc","properties":{"Name":{"type":"title","title":[{"plain_text":"T"}]}}}`)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ticket.Body, "Page 1") || !strings.Contains(ticket.Body, "Page 2") {
		t.Errorf("Body should contain both pages, got %q", ticket.Body)
	}
	if callCount != 2 {
		t.Errorf("expected 2 block API calls, got %d", callCount)
	}
}

func TestBlocksToMarkdown(t *testing.T) {
	blocks := []notionBlock{
		{Type: "paragraph", Paragraph: &blockContent{RichText: []richTextEntry{{PlainText: "Hello"}}}},
		{Type: "heading_1", Heading1: &blockContent{RichText: []richTextEntry{{PlainText: "Title"}}}},
		{Type: "heading_2", Heading2: &blockContent{RichText: []richTextEntry{{PlainText: "Subtitle"}}}},
		{Type: "heading_3", Heading3: &blockContent{RichText: []richTextEntry{{PlainText: "Section"}}}},
		{Type: "bulleted_list_item", BulletedListItem: &blockContent{RichText: []richTextEntry{{PlainText: "Item A"}}}},
		{Type: "bulleted_list_item", BulletedListItem: &blockContent{RichText: []richTextEntry{{PlainText: "Item B"}}}},
		{Type: "numbered_list_item", NumberedListItem: &blockContent{RichText: []richTextEntry{{PlainText: "First"}}}},
		{Type: "numbered_list_item", NumberedListItem: &blockContent{RichText: []richTextEntry{{PlainText: "Second"}}}},
		{Type: "to_do", ToDo: &toDoContent{RichText: []richTextEntry{{PlainText: "Done"}}, Checked: true}},
		{Type: "to_do", ToDo: &toDoContent{RichText: []richTextEntry{{PlainText: "Not done"}}, Checked: false}},
		{Type: "code", Code: &codeContent{RichText: []richTextEntry{{PlainText: "fmt.Println()"}}, Language: "go"}},
		{Type: "quote", Quote: &blockContent{RichText: []richTextEntry{{PlainText: "A wise quote"}}}},
		{Type: "divider"},
	}

	md := blocksToMarkdown(blocks)

	expected := []string{
		"Hello",
		"# Title",
		"## Subtitle",
		"### Section",
		"- Item A",
		"- Item B",
		"1. First",
		"2. Second",
		"- [x] Done",
		"- [ ] Not done",
		"```go\nfmt.Println()\n```",
		"> A wise quote",
		"---",
	}
	for _, exp := range expected {
		if !strings.Contains(md, exp) {
			t.Errorf("markdown should contain %q, got:\n%s", exp, md)
		}
	}
}

func TestRenderRichText(t *testing.T) {
	href := "https://example.com"
	entries := []richTextEntry{
		{PlainText: "bold", Annotations: &richTextAnnot{Bold: true}},
		{PlainText: " and ", Annotations: &richTextAnnot{}},
		{PlainText: "italic", Annotations: &richTextAnnot{Italic: true}},
		{PlainText: " and ", Annotations: &richTextAnnot{}},
		{PlainText: "code", Annotations: &richTextAnnot{Code: true}},
		{PlainText: " and ", Annotations: &richTextAnnot{}},
		{PlainText: "strike", Annotations: &richTextAnnot{Strikethrough: true}},
		{PlainText: " and ", Annotations: &richTextAnnot{}},
		{PlainText: "link", Href: &href},
	}

	result := renderRichText(entries)

	if !strings.Contains(result, "**bold**") {
		t.Errorf("result should contain **bold**, got %q", result)
	}
	if !strings.Contains(result, "*italic*") {
		t.Errorf("result should contain *italic*, got %q", result)
	}
	if !strings.Contains(result, "`code`") {
		t.Errorf("result should contain `code`, got %q", result)
	}
	if !strings.Contains(result, "~~strike~~") {
		t.Errorf("result should contain ~~strike~~, got %q", result)
	}
	if !strings.Contains(result, "[link](https://example.com)") {
		t.Errorf("result should contain [link](https://example.com), got %q", result)
	}
}

func TestNotionProvider_Fetch_NestedBlocks(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")

	toggleBlockID := "toggle-block-id"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/"+toggleBlockID+"/children") {
			// Children of the toggle block.
			resp := notionBlockList{
				Results: []notionBlock{
					{Type: "paragraph", Paragraph: &blockContent{RichText: []richTextEntry{{PlainText: "Nested content"}}}},
				},
				HasMore: false,
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(r.URL.Path, "/blocks/") {
			// Top-level blocks.
			resp := notionBlockList{
				Results: []notionBlock{
					{
						ID:          toggleBlockID,
						Type:        "toggle",
						HasChildren: true,
						Toggle:      &blockContent{RichText: []richTextEntry{{PlainText: "Toggle header"}}},
					},
				},
				HasMore: false,
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			_, _ = fmt.Fprint(w, `{"id":"abc","properties":{"Name":{"type":"title","title":[{"plain_text":"T"}]}}}`)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ticket.Body, "Toggle header") {
		t.Errorf("Body should contain toggle header, got %q", ticket.Body)
	}
	if !strings.Contains(ticket.Body, "Nested content") {
		t.Errorf("Body should contain nested content, got %q", ticket.Body)
	}
}

func TestNotionProvider_Fetch_CustomTitleProperty(t *testing.T) {
	t.Setenv("NOTION_API_KEY", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/blocks/") {
			_, _ = fmt.Fprint(w, `{"results":[],"has_more":false}`)
		} else {
			// Title property is "Task" instead of "Name".
			_, _ = fmt.Fprint(w, `{"id":"abc","properties":{"Task":{"type":"title","title":[{"plain_text":"Custom Title"}]},"Status":{"type":"select"}}}`)
		}
	}))
	defer server.Close()

	p := &NotionProvider{apiBase: server.URL}
	ticket, err := p.Fetch("https://notion.so/abc123de1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Custom Title")
	}
}

func TestBlocksToMarkdown_UnknownBlockType(t *testing.T) {
	blocks := []notionBlock{
		{Type: "paragraph", Paragraph: &blockContent{RichText: []richTextEntry{{PlainText: "Known"}}}},
		{Type: "image"},     // unknown/unsupported — should be skipped
		{Type: "embed"},     // unknown/unsupported — should be skipped
		{Type: "paragraph", Paragraph: &blockContent{RichText: []richTextEntry{{PlainText: "Also known"}}}},
	}

	md := blocksToMarkdown(blocks)
	if !strings.Contains(md, "Known") || !strings.Contains(md, "Also known") {
		t.Errorf("known blocks should be rendered, got %q", md)
	}
	if strings.Contains(md, "image") || strings.Contains(md, "embed") {
		t.Errorf("unknown blocks should be skipped, got %q", md)
	}
}
