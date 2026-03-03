package detect

import (
	"github.com/nqode/qode/internal/config"
)

// PythonDetector identifies a Python project.
type PythonDetector struct{}

func (d *PythonDetector) Name() string { return "python" }

func (d *PythonDetector) Detect(root string) (bool, float64) {
	if fileExists(root, "pyproject.toml") {
		return true, 1.0
	}
	if fileExists(root, "setup.py") || fileExists(root, "setup.cfg") {
		return true, 0.95
	}
	if fileExists(root, "requirements.txt") {
		return true, 0.8
	}
	if fileExists(root, "Pipfile") || fileExists(root, "Pipfile.lock") {
		return true, 0.9
	}
	if globExists(root, "*.py") {
		return true, 0.6
	}
	return false, 0
}

func (d *PythonDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["python"]
}
