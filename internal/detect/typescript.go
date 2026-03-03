package detect

import (
	"github.com/nqode/qode/internal/config"
)

// TypeScriptDetector identifies a plain TypeScript project
// (no framework — acts as a fallback for shared-libs layers).
type TypeScriptDetector struct{}

func (d *TypeScriptDetector) Name() string { return "typescript" }

func (d *TypeScriptDetector) Detect(root string) (bool, float64) {
	if fileExists(root, "tsconfig.json") {
		// Only claim if there is no framework-specific config present,
		// since those detectors are more specific.
		if fileExists(root, "angular.json") {
			return false, 0
		}
		// If package.json also exists with react/next/angular, don't claim.
		return true, 0.7
	}
	return false, 0
}

func (d *TypeScriptDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["typescript"]
}
