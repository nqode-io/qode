package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
)

func setupKBDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	kbDir := filepath.Join(root, ".qode", "knowledge")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return root
}

func writeTempFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func minimalConfig() *config.Config {
	return &config.Config{}
}

// --- listDir ---

func TestListDir_Recursive(t *testing.T) {
	root := setupKBDir(t)
	kbDir := filepath.Join(root, ".qode", "knowledge")

	writeTempFile(t, filepath.Join(kbDir, "top-level.md"), "top level")
	writeTempFile(t, filepath.Join(kbDir, "lessons", "lesson1.md"), "lesson 1")
	writeTempFile(t, filepath.Join(kbDir, "lessons", "lesson2.md"), "lesson 2")

	files, err := listDir(kbDir)
	if err != nil {
		t.Fatalf("listDir: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("listDir: got %d files, want 3", len(files))
	}
}

func TestListDir_SkipsHiddenDirs(t *testing.T) {
	root := setupKBDir(t)
	kbDir := filepath.Join(root, ".qode", "knowledge")

	writeTempFile(t, filepath.Join(kbDir, "visible.md"), "visible")
	writeTempFile(t, filepath.Join(kbDir, ".hidden", "secret.md"), "hidden")

	files, err := listDir(kbDir)
	if err != nil {
		t.Fatalf("listDir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("listDir: got %d files, want 1", len(files))
	}
}

func TestListDir_EmptyDir(t *testing.T) {
	root := setupKBDir(t)
	kbDir := filepath.Join(root, ".qode", "knowledge")

	files, err := listDir(kbDir)
	if err != nil {
		t.Fatalf("listDir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("listDir: got %d files, want 0", len(files))
	}
}

// --- Load ---

func TestLoad_LessonFilesNoDoubleHeader(t *testing.T) {
	root := setupKBDir(t)
	lessonsDir := filepath.Join(root, ".qode", "knowledge", "lessons")
	writeTempFile(t, filepath.Join(lessonsDir, "my-lesson.md"), "### My Lesson\nSome content here.")

	output, err := Load(root, minimalConfig())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Should contain the lesson's own heading
	if !strings.Contains(output, "### My Lesson") {
		t.Error("Load: missing lesson heading")
	}
	// Should NOT contain a filename header like "### my-lesson.md"
	if strings.Contains(output, "### my-lesson.md") {
		t.Error("Load: lesson file has double header (### my-lesson.md)")
	}
}

func TestLoad_RegularFilesKeepHeader(t *testing.T) {
	root := setupKBDir(t)
	kbDir := filepath.Join(root, ".qode", "knowledge")
	writeTempFile(t, filepath.Join(kbDir, "guide.md"), "Some guide content.")

	output, err := Load(root, minimalConfig())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(output, "### guide.md") {
		t.Error("Load: regular file missing ### filename header")
	}
}

func TestLoad_MixedFiles(t *testing.T) {
	root := setupKBDir(t)
	kbDir := filepath.Join(root, ".qode", "knowledge")
	writeTempFile(t, filepath.Join(kbDir, "regular.md"), "Regular content.")
	writeTempFile(t, filepath.Join(kbDir, "lessons", "lesson.md"), "### A Lesson\nLesson body.")

	output, err := Load(root, minimalConfig())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(output, "### regular.md") {
		t.Error("Load: missing header for regular file")
	}
	if strings.Contains(output, "### lesson.md") {
		t.Error("Load: lesson file should not have filename header")
	}
	if !strings.Contains(output, "### A Lesson") {
		t.Error("Load: missing lesson content heading")
	}
}

// --- ToKebabCase ---

func TestToKebabCase_Spaces(t *testing.T) {
	got := ToKebabCase("Hello World")
	if got != "hello-world" {
		t.Errorf("ToKebabCase(%q) = %q, want %q", "Hello World", got, "hello-world")
	}
}

func TestToKebabCase_SpecialChars(t *testing.T) {
	got := ToKebabCase("Don't use this!")
	if got != "don-t-use-this" {
		t.Errorf("ToKebabCase(%q) = %q, want %q", "Don't use this!", got, "don-t-use-this")
	}
}

func TestToKebabCase_ConsecutiveHyphens(t *testing.T) {
	got := ToKebabCase("foo---bar")
	if got != "foo-bar" {
		t.Errorf("ToKebabCase(%q) = %q, want %q", "foo---bar", got, "foo-bar")
	}
}

func TestToKebabCase_NonASCII(t *testing.T) {
	got := ToKebabCase("über cool")
	// Non-ASCII bytes get replaced with hyphens, collapsed
	if !strings.HasSuffix(got, "ber-cool") {
		t.Errorf("ToKebabCase(%q) = %q, want suffix %q", "über cool", got, "ber-cool")
	}
}

// --- SaveLesson ---

func TestSaveLesson_WritesCorrectFile(t *testing.T) {
	root := t.TempDir()
	content := "### My Lesson\nSome body."
	if err := SaveLesson(root, "My Lesson", content); err != nil {
		t.Fatalf("SaveLesson: %v", err)
	}

	path := filepath.Join(LessonsDir(root), "my-lesson.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != content {
		t.Errorf("SaveLesson content = %q, want %q", string(data), content)
	}
}

func TestSaveLesson_CreatesDir(t *testing.T) {
	root := t.TempDir()
	// lessons/ dir does not exist yet
	if err := SaveLesson(root, "New Lesson", "content"); err != nil {
		t.Fatalf("SaveLesson should create dir: %v", err)
	}

	path := filepath.Join(LessonsDir(root), "new-lesson.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

func TestSaveLesson_HandlesCollision(t *testing.T) {
	root := t.TempDir()
	if err := SaveLesson(root, "Same Title", "first"); err != nil {
		t.Fatalf("SaveLesson first: %v", err)
	}
	if err := SaveLesson(root, "Same Title", "second"); err != nil {
		t.Fatalf("SaveLesson second: %v", err)
	}

	dir := LessonsDir(root)
	first, err := os.ReadFile(filepath.Join(dir, "same-title.md"))
	if err != nil {
		t.Fatalf("ReadFile first: %v", err)
	}
	if string(first) != "first" {
		t.Errorf("first file content = %q, want %q", string(first), "first")
	}

	second, err := os.ReadFile(filepath.Join(dir, "same-title-2.md"))
	if err != nil {
		t.Fatalf("ReadFile second: %v", err)
	}
	if string(second) != "second" {
		t.Errorf("second file content = %q, want %q", string(second), "second")
	}
}

// --- ListLessons ---

func TestListLessons_ParsesTitleAndSummary(t *testing.T) {
	root := t.TempDir()
	content := "### Validate input early\nAlways validate user input before passing it to the database layer.\n\n**Example 1:** Bad\n```go\ndb.Query(input)\n```"
	if err := SaveLesson(root, "Validate input early", content); err != nil {
		t.Fatalf("SaveLesson: %v", err)
	}

	lessons, err := ListLessons(root)
	if err != nil {
		t.Fatalf("ListLessons: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("ListLessons: got %d, want 1", len(lessons))
	}
	if lessons[0].Title != "Validate input early" {
		t.Errorf("Title = %q, want %q", lessons[0].Title, "Validate input early")
	}
	if lessons[0].Summary != "Always validate user input before passing it to the database layer." {
		t.Errorf("Summary = %q, want %q", lessons[0].Summary, "Always validate user input before passing it to the database layer.")
	}
}

func TestListLessons_EmptyDir(t *testing.T) {
	root := t.TempDir()
	lessons, err := ListLessons(root)
	if err != nil {
		t.Fatalf("ListLessons: %v", err)
	}
	if lessons != nil {
		t.Errorf("ListLessons on empty dir: got %v, want nil", lessons)
	}
}

// --- parseLessonHeader ---

func TestParseLessonHeader_NoHeading(t *testing.T) {
	title, summary := parseLessonHeader("Just some text without a heading.")
	if title != "" {
		t.Errorf("title = %q, want empty", title)
	}
	if summary != "" {
		t.Errorf("summary = %q, want empty", summary)
	}
}

func TestParseLessonHeader_EmptyContent(t *testing.T) {
	title, summary := parseLessonHeader("")
	if title != "" {
		t.Errorf("title = %q, want empty", title)
	}
	if summary != "" {
		t.Errorf("summary = %q, want empty", summary)
	}
}

// --- isLessonFile ---

func TestIsLessonFile(t *testing.T) {
	if !isLessonFile("/root/.qode/knowledge/lessons/my-lesson.md") {
		t.Error("expected lesson file to be detected")
	}
	if isLessonFile("/root/.qode/knowledge/guide.md") {
		t.Error("expected non-lesson file to not be detected")
	}
}

// --- Integration: List includes lessons ---

func TestKnowledgeList_IncludesLessons(t *testing.T) {
	root := setupKBDir(t)
	lessonsDir := filepath.Join(root, ".qode", "knowledge", "lessons")
	writeTempFile(t, filepath.Join(lessonsDir, "test-lesson.md"), "### Test\nBody.")

	files, err := List(root, minimalConfig())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, f := range files {
		if strings.HasSuffix(f, "test-lesson.md") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List: lesson file not found in results: %v", files)
	}
}

// --- Integration: Search finds in lessons ---

func TestKnowledgeSearch_FindsInLessons(t *testing.T) {
	root := setupKBDir(t)
	lessonsDir := filepath.Join(root, ".qode", "knowledge", "lessons")
	writeTempFile(t, filepath.Join(lessonsDir, "goroutine-leak.md"), "### Avoid goroutine leaks\nUse errgroup for lifecycle management.")

	results, err := Search(root, minimalConfig(), "goroutine")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("Search: expected to find goroutine in lesson file")
	}
}
