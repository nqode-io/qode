package detect

import (
	"github.com/nqode/qode/internal/config"
)

// AngularDetector identifies an Angular project.
type AngularDetector struct{}

func (d *AngularDetector) Name() string { return "angular" }

func (d *AngularDetector) Detect(root string) (bool, float64) {
	// angular.json is the definitive signal.
	if fileExists(root, "angular.json") {
		return true, 1.0
	}
	// .angular/ cache directory.
	if fileExists(root, ".angular") {
		return true, 0.9
	}
	return false, 0
}

func (d *AngularDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["angular"]
}
