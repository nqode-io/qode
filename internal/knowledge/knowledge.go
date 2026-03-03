package knowledge

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
)

// SearchResult is a snippet match from the knowledge base.
type SearchResult struct {
	File    string
	Snippet string
}

// Load reads all knowledge base files and returns them concatenated for
// prompt injection.
func Load(root string, cfg *config.Config) (string, error) {
	files, err := List(root, cfg)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		sb.WriteString("### ")
		sb.WriteString(filepath.Base(f))
		sb.WriteString("\n\n")
		sb.Write(data)
		sb.WriteString("\n\n---\n\n")
	}
	return sb.String(), nil
}

// List returns all knowledge base file paths.
func List(root string, cfg *config.Config) ([]string, error) {
	var files []string

	// Explicit paths from config.
	for _, p := range cfg.Knowledge.Paths {
		abs := filepath.Join(root, p)
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if info.IsDir() {
			dirFiles, _ := listDir(abs)
			files = append(files, dirFiles...)
		} else {
			files = append(files, abs)
		}
	}

	// Auto-discover from .qode/knowledge/.
	kbDir := filepath.Join(root, config.QodeDir, "knowledge")
	kbFiles, _ := listDir(kbDir)
	files = append(files, kbFiles...)

	return dedup(files), nil
}

// Search searches knowledge base files for a query string.
func Search(root string, cfg *config.Config, query string) ([]SearchResult, error) {
	files, err := List(root, cfg)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []SearchResult

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), query) {
				snippet := strings.TrimSpace(line)
				if len(snippet) > 120 {
					snippet = snippet[:120] + "..."
				}
				results = append(results, SearchResult{
					File:    f,
					Snippet: snippet,
				})
				if len(results) >= 20 {
					return results, nil
				}
				break
			}
		}
	}
	return results, nil
}

func listDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func dedup(files []string) []string {
	seen := map[string]bool{}
	result := files[:0]
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	return result
}
