package detect

import (
	"github.com/nqode/qode/internal/config"
)

// GoDetector identifies a Go project.
type GoDetector struct{}

func (d *GoDetector) Name() string { return "go" }

func (d *GoDetector) Detect(root string) (bool, float64) {
	if fileExists(root, "go.mod") {
		return true, 1.0
	}
	if globExists(root, "*.go") {
		return true, 0.7
	}
	return false, 0
}

func (d *GoDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["go"]
}
