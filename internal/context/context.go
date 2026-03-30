package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/scoring"
)

// Iteration holds metadata about one refinement pass.
type Iteration struct {
	Number int
	Score  int
	File   string
}

// Context holds the contents of the branch context folder.
type Context struct {
	Branch     string
	ContextDir string

	// Content files.
	Ticket  string
	Mockups []string // paths to image files
	Extra   []string // other text file contents

	// Refinement history.
	Iterations []Iteration

	// Derived.
	RefinedAnalysis string // most recent refined-analysis.md
	Spec            string // spec.md content
}

// Load reads the context folder for a branch.
func Load(root, branch string) (*Context, error) {
	dir := filepath.Join(root, config.QodeDir, "branches", branch)
	ctx := &Context{
		Branch:     branch,
		ContextDir: dir,
	}

	// Context sub-directory.
	ctxSubDir := filepath.Join(dir, "context")
	_ = os.MkdirAll(ctxSubDir, 0755)

	ctx.Ticket = readFileOr(filepath.Join(ctxSubDir, "ticket.md"), "")

	// Scan for extra context files.
	entries, _ := os.ReadDir(ctxSubDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		switch name {
		case "ticket.md", "notes.md":
			continue
		}
		full := filepath.Join(ctxSubDir, name)
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			ctx.Mockups = append(ctx.Mockups, full)
		default:
			content := readFileOr(full, "")
			if content != "" {
				ctx.Extra = append(ctx.Extra, "### "+name+"\n\n"+content)
			}
		}
	}

	// Load refinement iterations.
	iterFiles, _ := filepath.Glob(filepath.Join(dir, "refined-analysis-*.md"))
	for _, f := range iterFiles {
		base := filepath.Base(f)
		// Expect name like "refined-analysis-1.md" or "refined-analysis-1-score-22.md".
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
		ctx.Iterations = append(ctx.Iterations, Iteration{Number: num, Score: score, File: f})
	}

	// Merge iteration from refined-analysis.md header if newer than glob results.
	if n, score, ok := parseIterationFromAnalysis(dir); ok {
		if n > maxIterationNumber(ctx.Iterations) {
			ctx.Iterations = append(ctx.Iterations, Iteration{Number: n, Score: score})
		}
	}

	// Latest refined analysis.
	latestAnalysis := filepath.Join(dir, "refined-analysis.md")
	ctx.RefinedAnalysis = readFileOr(latestAnalysis, "")

	// Spec.
	ctx.Spec = readFileOr(filepath.Join(dir, "spec.md"), "")

	return ctx, nil
}

// HasSpec returns true if a spec file exists.
func (c *Context) HasSpec() bool { return c.Spec != "" }

// HasRefinedAnalysis returns true if a refined analysis exists.
func (c *Context) HasRefinedAnalysis() bool { return c.RefinedAnalysis != "" }

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

// HasCodeReview returns true if a code-review.md exists in the branch context.
func (c *Context) HasCodeReview() bool {
	_, err := os.Stat(filepath.Join(c.ContextDir, "code-review.md"))
	return err == nil
}

// HasSecurityReview returns true if a security-review.md exists in the branch context.
func (c *Context) HasSecurityReview() bool {
	_, err := os.Stat(filepath.Join(c.ContextDir, "security-review.md"))
	return err == nil
}

// CodeReviewScore returns the total score from code-review.md, or 0 if absent or unparseable.
func (c *Context) CodeReviewScore() float64 {
	return scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "code-review.md"))
}

// SecurityReviewScore returns the total score from security-review.md, or 0 if absent or unparseable.
func (c *Context) SecurityReviewScore() float64 {
	return scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "security-review.md"))
}

func readFileOr(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(data)
}

func parseIterationFromAnalysis(branchDir string) (int, int, bool) {
	data, err := os.ReadFile(filepath.Join(branchDir, "refined-analysis.md"))
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
