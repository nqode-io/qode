package knowledge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
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
		if !isLessonFile(f) {
			sb.WriteString("### ")
			sb.WriteString(filepath.Base(f))
			sb.WriteString("\n\n")
		}
		sb.Write(data)
		sb.WriteString("\n\n---\n\n")
	}
	return sb.String(), nil
}

// List returns all knowledge base file paths.
func List(root string, cfg *config.Config) ([]string, error) {
	var files []string

	kbPath := cfg.Knowledge.Path
	if kbPath == "" {
		kbPath = filepath.Join(config.QodeDir, "knowledge")
	}
	kbDir := filepath.Join(root, kbPath)
	kbFiles, _ := listDir(kbDir)
	files = append(files, kbFiles...)

	return files, nil
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
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// LessonSummary holds parsed metadata from a lesson file.
type LessonSummary struct {
	Title   string
	Path    string
	Summary string
}

// LessonsDir returns the path to the lessons subdirectory.
func LessonsDir(root string) string {
	return filepath.Join(root, config.QodeDir, "knowledge", "lessons")
}

// ListLessons returns summaries of all existing lesson files.
func ListLessons(root string) ([]LessonSummary, error) {
	dir := LessonsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var lessons []LessonSummary
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, e.Name()))
		if readErr != nil {
			continue
		}
		title, summary := parseLessonHeader(string(data))
		lessons = append(lessons, LessonSummary{
			Title:   title,
			Path:    filepath.Join(dir, e.Name()),
			Summary: summary,
		})
	}
	return lessons, nil
}

// SaveLesson writes a lesson file with a kebab-case filename derived from title.
// If a file with the same name exists, it appends a numeric suffix.
func SaveLesson(root, title, content string) error {
	dir := LessonsDir(root)
	if err := iokit.EnsureDir(dir); err != nil {
		return err
	}
	base := ToKebabCase(title) + ".md"
	dest := filepath.Join(dir, base)
	for i := 2; fileExists(dest); i++ {
		dest = filepath.Join(dir, fmt.Sprintf("%s-%d.md", ToKebabCase(title), i))
	}
	return iokit.WriteFile(dest, []byte(content), 0644)
}

// ToKebabCase converts a string to a kebab-case filename component.
func ToKebabCase(s string) string {
	s = strings.ToLower(s)
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	parts := strings.FieldsFunc(string(result), func(r rune) bool { return r == '-' })
	return strings.Join(parts, "-")
}

func parseLessonHeader(content string) (title, summary string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if title == "" {
			if strings.HasPrefix(trimmed, "### ") {
				title = strings.TrimPrefix(trimmed, "### ")
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "**Example") {
			break
		}
		summary = trimmed
		break
	}
	return title, summary
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isLessonFile(path string) bool {
	return strings.Contains(path, string(filepath.Separator)+"lessons"+string(filepath.Separator))
}

