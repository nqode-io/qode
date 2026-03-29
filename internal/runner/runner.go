package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/scoring"
)

// GateResult is the outcome of a single quality gate.
type GateResult struct {
	Passed  bool
	Skipped bool
	Detail  string
}

// LayerResult aggregates quality gate results for one layer.
type LayerResult struct {
	Layer          string
	Stack          string
	Tests          GateResult
	Lint           GateResult
	CodeReview     float64
	SecurityReview float64
	Passed         bool
}

// RunCheck executes quality gates for all given layers.
// branch is the current git branch name, used to locate saved review files.
func RunCheck(root, branch string, cfg *config.Config, layers []config.LayerConfig) []LayerResult {
	results := make([]LayerResult, 0, len(layers))
	for _, l := range layers {
		r := runLayerCheck(root, branch, cfg, l)
		results = append(results, r)
	}
	return results
}

func runLayerCheck(root, branch string, cfg *config.Config, layer config.LayerConfig) LayerResult {
	layerRoot := filepath.Join(root, layer.Path)
	result := LayerResult{
		Layer: layer.Name,
		Stack: layer.Stack,
	}

	// Tests.
	if layer.Test.Unit == "" {
		result.Tests = GateResult{Skipped: true, Detail: "no test command"}
	} else {
		result.Tests = runCommand(layerRoot, layer.Test.Unit)
	}

	// Lint.
	if layer.Test.Lint == "" {
		result.Lint = GateResult{Skipped: true, Detail: "no lint command"}
	} else {
		result.Lint = runCommand(layerRoot, layer.Test.Lint)
	}

	// Reviews — read scores from saved review files for the current branch.
	// A score of 0 means "not yet run" and is treated as skipped (not blocking).
	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)
	result.CodeReview = scoring.ExtractScoreFromFile(filepath.Join(branchDir, "code-review.md"))
	result.SecurityReview = scoring.ExtractScoreFromFile(filepath.Join(branchDir, "security-review.md"))

	codeOK := result.CodeReview == 0 || result.CodeReview >= cfg.Review.MinCodeScore
	secOK := result.SecurityReview == 0 || result.SecurityReview >= cfg.Review.MinSecurityScore

	result.Passed = (result.Tests.Passed || result.Tests.Skipped) &&
		(result.Lint.Passed || result.Lint.Skipped) &&
		codeOK && secOK

	return result
}

// runCommand executes a shell command and returns a GateResult.
func runCommand(dir, command string) GateResult {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return GateResult{Skipped: true}
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir

	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		last := lines[len(lines)-1]
		if len(last) > 60 {
			last = last[:60] + "..."
		}
		return GateResult{Passed: false, Detail: fmt.Sprintf("FAILED (%s): %s", elapsed, last)}
	}

	return GateResult{Passed: true, Detail: elapsed.String()}
}
