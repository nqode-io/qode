// Package qodecontext manages per-context work directories under .qode/contexts/<name>/.
// A context is a named work unit independent of any VCS; the active context is
// selected via a "current" symlink.
package qodecontext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/scoring"
)

// ErrNoCurrentContext is returned when no active context symlink exists.
var ErrNoCurrentContext = errors.New("no active context")

// ErrDanglingSymlink is returned when the "current" symlink points to a non-existent directory.
var ErrDanglingSymlink = errors.New("current context symlink is dangling")

const (
	contextsDir   = "contexts"
	currentLink   = "current"
	ctxNameFile   = ".ctx-name.md"
	ticketStub    = "# Ticket\n\nPaste ticket content here, or run the `qode-ticket-fetch` step in your IDE.\n"
	notesStub     = "# Notes\n\nAdd any additional context, decisions, or open questions here.\n"
	maxNameLength = 255
)

// Iteration holds metadata about one refinement pass.
type Iteration struct {
	Number int
	Score  int
	File   string
}

// Context holds the state for a named context directory.
type Context struct {
	ContextName string
	ContextDir  string

	// Content files.
	Ticket  string
	Mockups []string // paths to image files

	// Refinement history.
	Iterations []Iteration

	// Derived.
	RefinedAnalysis string // most recent refined-analysis.md
	Spec            string // spec.md content
}

// Load reads the active context (via the "current" symlink).
func Load(ctx context.Context, root string) (*Context, error) {
	name, err := CurrentName(ctx, root)
	if err != nil {
		return nil, err
	}
	return LoadByName(ctx, root, name)
}

// LoadByName reads the context directory for the named context.
func LoadByName(ctx context.Context, root, name string) (*Context, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := safeContextDir(root, name)
	if err != nil {
		return nil, err
	}

	qctx := &Context{
		ContextName: name,
		ContextDir:  dir,
	}

	qctx.Ticket = iokit.ReadFileOrString(filepath.Join(dir, "ticket.md"), "")

	// Scan for mockup images.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			qctx.Mockups = append(qctx.Mockups, filepath.Join(dir, e.Name()))
		}
	}

	// Load refinement iterations.
	iterFiles, _ := filepath.Glob(filepath.Join(dir, "refined-analysis-*.md"))
	for _, f := range iterFiles {
		base := filepath.Base(f)
		parts := strings.Split(strings.TrimSuffix(base, ".md"), "-")
		num := 0
		score := 0
		for i, p := range parts {
			if p == "analysis" && i+1 < len(parts) {
				num, _ = strconv.Atoi(parts[i+1])
			}
			if p == "score" && i+1 < len(parts) {
				score, _ = strconv.Atoi(parts[i+1])
			}
		}
		qctx.Iterations = append(qctx.Iterations, Iteration{Number: num, Score: score, File: f})
	}

	// Merge iteration from refined-analysis.md header if newer.
	if n, score, ok := parseIterationFromAnalysis(dir); ok {
		if n > maxIterationNumber(qctx.Iterations) {
			qctx.Iterations = append(qctx.Iterations, Iteration{Number: n, Score: score})
		}
	}

	qctx.RefinedAnalysis = iokit.ReadFileOrString(filepath.Join(dir, "refined-analysis.md"), "")
	qctx.Spec = iokit.ReadFileOrString(filepath.Join(dir, "spec.md"), "")

	return qctx, nil
}

// Init creates a new context directory with stub files.
func Init(ctx context.Context, root, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateContextName(name); err != nil {
		return err
	}

	dir, err := safeContextDir(root, name)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(dir); statErr == nil {
		return fmt.Errorf("context %q already exists", name)
	}

	if err := iokit.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating context directory: %w", err)
	}

	if err := iokit.WriteFile(filepath.Join(dir, ctxNameFile), []byte(name), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", ctxNameFile, err)
	}
	if err := iokit.WriteFile(filepath.Join(dir, "ticket.md"), []byte(ticketStub), 0644); err != nil {
		return fmt.Errorf("writing ticket.md: %w", err)
	}
	if err := iokit.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesStub), 0644); err != nil {
		return fmt.Errorf("writing notes.md: %w", err)
	}

	return nil
}

// Switch atomically replaces the "current" symlink to point to name.
func Switch(ctx context.Context, root, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateContextName(name); err != nil {
		return err
	}

	dir, err := safeContextDir(root, name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("context %q not found", name)
	}

	base := filepath.Join(root, config.QodeDir, contextsDir)
	if err := iokit.EnsureDir(base); err != nil {
		return err
	}

	link := filepath.Join(base, currentLink)
	tmp := link + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(name, tmp); err != nil {
		return iokit.WrapSymlinkError(err)
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("switching context: %w", err)
	}

	return nil
}

// Clear removes all files except .ctx-name.md and reinitialises ticket.md and notes.md.
func Clear(ctx context.Context, root, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		var err error
		name, err = CurrentName(ctx, root)
		if err != nil {
			return err
		}
	} else {
		if err := ValidateContextName(name); err != nil {
			return err
		}
	}

	dir, err := safeContextDir(root, name)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading context directory: %w", err)
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.Name() == ctxNameFile {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name(), err)
		}
	}

	if err := iokit.WriteFile(filepath.Join(dir, "ticket.md"), []byte(ticketStub), 0644); err != nil {
		return fmt.Errorf("reinitialising ticket.md: %w", err)
	}
	if err := iokit.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesStub), 0644); err != nil {
		return fmt.Errorf("reinitialising notes.md: %w", err)
	}

	return nil
}

// Remove deletes a context directory. If "current" points to it, the symlink
// is removed first.
func Remove(ctx context.Context, root, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		var err error
		name, err = CurrentName(ctx, root)
		if err != nil {
			return err
		}
	} else {
		if err := ValidateContextName(name); err != nil {
			return err
		}
	}

	dir, err := safeContextDir(root, name)
	if err != nil {
		return err
	}

	// Remove "current" symlink if it points to this context.
	link := filepath.Join(root, config.QodeDir, contextsDir, currentLink)
	if target, readErr := os.Readlink(link); readErr == nil && target == name {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("removing current symlink: %w", err)
		}
	}

	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing context directory: %w", err)
	}

	return nil
}

// Reset removes the "current" symlink if it exists.
func Reset(ctx context.Context, root string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	link := filepath.Join(root, config.QodeDir, contextsDir, currentLink)
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing current symlink: %w", err)
	}
	return nil
}

// CurrentName resolves the name of the active context from the "current" symlink.
func CurrentName(ctx context.Context, root string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	link := filepath.Join(root, config.QodeDir, contextsDir, currentLink)

	target, err := os.Readlink(link)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w — run 'qode context init <name>' or 'qode context switch <name>'", ErrNoCurrentContext)
		}
		return "", fmt.Errorf("reading current symlink: %w", err)
	}

	// Validate the target directory exists (detect dangling symlinks).
	dir, err := safeContextDir(root, target)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("%w — run 'qode context reset' to clear it", ErrDanglingSymlink)
	}

	return target, nil
}

// SanitizeContextName replaces "/" with "--" for filesystem safety.
func SanitizeContextName(name string) string {
	return strings.ReplaceAll(name, "/", "--")
}

// ValidateContextName returns an error if name is not a safe context name.
func ValidateContextName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid context name: must not be empty")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("invalid context name: exceeds %d characters", maxNameLength)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid context name: %q is reserved", name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid context name: must not contain path separators")
	}
	return nil
}

// safeContextDir computes and validates the absolute path for a context,
// preventing path traversal attacks.
func safeContextDir(root, name string) (string, error) {
	base := filepath.Join(root, config.QodeDir, contextsDir)
	target := filepath.Join(base, name)
	rel, err := filepath.Rel(base, target)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", fmt.Errorf("invalid context name %q: path traversal detected", name)
	}
	return target, nil
}

// HasSpec returns true if a spec file exists.
func (c *Context) HasSpec() bool { return c.Spec != "" }

// HasRefinedAnalysis returns true if a refined analysis exists.
func (c *Context) HasRefinedAnalysis() bool { return c.RefinedAnalysis != "" }

// NextIteration returns the next iteration number.
func (c *Context) NextIteration() int {
	return maxIterationNumber(c.Iterations) + 1
}

// LatestScore returns the score from the most recent iteration, or 0.
func (c *Context) LatestScore() int {
	latestNum := 0
	latestScore := 0
	for _, it := range c.Iterations {
		if it.Number > latestNum {
			latestNum = it.Number
			latestScore = it.Score
		}
	}
	return latestScore
}

// HasCodeReview returns true if code-review.md exists.
func (c *Context) HasCodeReview() bool {
	_, err := os.Stat(filepath.Join(c.ContextDir, "code-review.md"))
	return err == nil
}

// HasSecurityReview returns true if security-review.md exists.
func (c *Context) HasSecurityReview() bool {
	_, err := os.Stat(filepath.Join(c.ContextDir, "security-review.md"))
	return err == nil
}

// CodeReviewScore returns the total score from code-review.md, or 0.
func (c *Context) CodeReviewScore() float64 {
	return scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "code-review.md"))
}

// SecurityReviewScore returns the total score from security-review.md, or 0.
func (c *Context) SecurityReviewScore() float64 {
	return scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "security-review.md"))
}

func parseIterationFromAnalysis(contextDir string) (int, int, bool) {
	data, err := os.ReadFile(filepath.Join(contextDir, "refined-analysis.md"))
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.SplitN(string(data), "\n", 6) {
		var n, score, total int
		if _, err := fmt.Sscanf(line, "<!-- qode:iteration=%d score=%d/%d -->", &n, &score, &total); err == nil {
			return n, score, true
		}
	}
	return 0, 0, false
}

func maxIterationNumber(iterations []Iteration) int {
	max := 0
	for _, it := range iterations {
		if it.Number > max {
			max = it.Number
		}
	}
	return max
}
