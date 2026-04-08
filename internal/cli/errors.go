package cli

import "errors"

// Sentinel errors for common CLI failure conditions.
var (
	// ErrNotInitialised is returned when qode.yaml or project structure is missing.
	ErrNotInitialised = errors.New("project not initialised: run 'qode init'")
	// ErrNoAnalysis is returned when refined-analysis.md is required but absent.
	ErrNoAnalysis = errors.New("no refined analysis")
	// ErrNoSpec is returned when spec.md is required but absent.
	ErrNoSpec = errors.New("no spec")
	// ErrNoChanges is returned when a review is requested but no diff exists.
	ErrNoChanges = errors.New("no changes detected: commit code first before running a review")
)
