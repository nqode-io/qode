// Command qode-loop is the bookkeeping helper for this repository's serial
// ticket loop (see docs/qode-loop.md). It is contributor tooling, not a qode
// feature: it lives outside internal/, imports no internal packages, and
// exists so the orchestrating agent runs a stable, whitelistable command
// instead of ad-hoc inline scripts.
//
// Usage:
//
//	go run ./tools/qode-loop state init ticket=72 [key=value ...]
//	go run ./tools/qode-loop state get [key]
//	go run ./tools/qode-loop state set key=value [key=value ...]
//	go run ./tools/qode-loop refine-score <iteration> <score> <max>
//
// Paths are resolved relative to the working directory, which must be the
// repository root. Values given as key=value are stored as integers when
// they are all digits, as null when written "null", and as strings otherwise.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	contextDir   = ".qode/contexts/current"
	stateFile    = "loop-state.json"
	analysisFile = "refined-analysis.md"
	headerPrefix = "<!-- qode:iteration="
	exitUsage    = 64
	filePerm     = 0o644
	minIteration = 1
	minMaxScore  = 1
)

// errUsage marks a command-line usage error; main maps it to exitUsage.
var errUsage = errors.New("usage: qode-loop state init|get|set [key=value ...] | refine-score <iteration> <score> <max>")

func main() {
	if err := run(os.Args[1:], os.Stdout, contextDir); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "qode-loop:", err)
		if errors.Is(err, errUsage) {
			os.Exit(exitUsage)
		}
		os.Exit(1)
	}
}

// run dispatches to the subcommand named by args[0].
func run(args []string, out io.Writer, dir string) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "state":
		return runState(args[1:], out, dir)
	case "refine-score":
		return runRefineScore(args[1:], out, dir)
	default:
		return fmt.Errorf("unknown command %q: %w", args[0], errUsage)
	}
}

// freshState is the state of a ticket that has not started any stage yet.
func freshState() map[string]any {
	return map[string]any{
		"ticket":              nil,
		"stage":               "setup",
		"refineIteration":     0,
		"codeReviewRound":     0,
		"securityReviewRound": 0,
		"prNumber":            nil,
	}
}

// runState handles `state init|get|set`.
func runState(args []string, out io.Writer, dir string) error {
	if len(args) == 0 {
		return errUsage
	}
	path := filepath.Join(dir, stateFile)
	switch args[0] {
	case "init":
		state := freshState()
		if err := applyPairs(state, args[1:]); err != nil {
			return err
		}
		return saveAndPrint(path, state, out)
	case "get":
		state, err := loadState(path)
		if err != nil {
			return err
		}
		return printState(state, args[1:], out)
	case "set":
		state, err := loadState(path)
		if err != nil {
			return err
		}
		if err := applyPairs(state, args[1:]); err != nil {
			return err
		}
		return saveAndPrint(path, state, out)
	default:
		return fmt.Errorf("unknown state command %q: %w", args[0], errUsage)
	}
}

// printState prints the whole state, or only the value of the key given as args[0].
func printState(state map[string]any, args []string, out io.Writer) error {
	var value any = state
	if len(args) > 0 {
		value = state[args[0]]
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	_, _ = fmt.Fprintln(out, string(data))
	return nil
}

func loadState(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return state, nil
}

func saveAndPrint(path string, state map[string]any, out io.Writer) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	_, _ = fmt.Fprintln(out, string(data))
	return nil
}

// applyPairs merges key=value arguments into state.
func applyPairs(state map[string]any, pairs []string) error {
	for _, pair := range pairs {
		key, raw, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return fmt.Errorf("expected key=value, got %q: %w", pair, errUsage)
		}
		state[key] = parseValue(raw)
	}
	return nil
}

// parseValue turns "null" into nil, all-digit strings into int, anything else into string.
func parseValue(raw string) any {
	if raw == "null" {
		return nil
	}
	if n, err := strconv.Atoi(raw); err == nil && n >= 0 && !strings.HasPrefix(raw, "+") {
		return n
	}
	return raw
}

// runRefineScore stamps the judge score into the analysis header and archives
// the scored iteration as refined-analysis-<iteration>-score-<score>.md.
func runRefineScore(args []string, out io.Writer, dir string) error {
	iteration, score, maxScore, err := parseScoreArgs(args)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, analysisFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	stamped := stampHeader(string(data), iteration, score, maxScore)
	if err := os.WriteFile(path, []byte(stamped), filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	archive := filepath.Join(dir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, score))
	if err := os.WriteFile(archive, []byte(stamped), filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", archive, err)
	}
	_, _ = fmt.Fprintf(out, "stamped iteration=%d score=%d/%d; archived %s\n", iteration, score, maxScore, archive)
	return nil
}

// parseScoreArgs validates <iteration> <score> <max>.
func parseScoreArgs(args []string) (iteration, score, maxScore int, err error) {
	const want = 3
	if len(args) != want {
		return 0, 0, 0, fmt.Errorf("refine-score takes %d arguments: %w", want, errUsage)
	}
	nums := make([]int, want)
	for i, arg := range args {
		n, convErr := strconv.Atoi(arg)
		if convErr != nil {
			return 0, 0, 0, fmt.Errorf("argument %q is not an integer: %w", arg, errUsage)
		}
		nums[i] = n
	}
	iteration, score, maxScore = nums[0], nums[1], nums[2]
	if iteration < minIteration || maxScore < minMaxScore || score < 0 || score > maxScore {
		return 0, 0, 0, fmt.Errorf("need iteration >= %d, max >= %d, 0 <= score <= max: %w", minIteration, minMaxScore, errUsage)
	}
	return iteration, score, maxScore, nil
}

// stampHeader replaces an existing qode iteration header on line 1, or
// prepends one when the analysis has none.
func stampHeader(content string, iteration, score, maxScore int) string {
	header := fmt.Sprintf("%s%d score=%d/%d -->", headerPrefix, iteration, score, maxScore)
	lines := strings.SplitN(content, "\n", 2)
	if strings.HasPrefix(lines[0], headerPrefix) {
		lines[0] = header
		return strings.Join(lines, "\n")
	}
	return header + "\n" + content
}
