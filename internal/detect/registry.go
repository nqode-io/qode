package detect

import "github.com/nqode/qode/internal/config"

// StackDetector identifies a technology stack from filesystem signals.
type StackDetector interface {
	// Name returns the canonical stack name (e.g. "react", "dotnet").
	Name() string
	// Detect checks root for signals and returns whether the stack is present
	// and a confidence score between 0.0 and 1.0.
	Detect(root string) (bool, float64)
	// DefaultConfig returns the default test/build commands for this stack.
	DefaultConfig() config.TestConfig
}

// DetectedLayer is a stack detected at a specific path.
type DetectedLayer struct {
	Name       string
	Path       string
	Stack      string
	Confidence float64
	Test       config.TestConfig
}

var registry []StackDetector

// Register adds a detector to the global registry.
func Register(d StackDetector) {
	registry = append(registry, d)
}

// All returns all registered detectors.
func All() []StackDetector {
	return registry
}

func init() {
	Register(&NextJSDetector{})
	Register(&ReactDetector{})
	Register(&DotNetDetector{})
	Register(&AngularDetector{})
	Register(&JavaDetector{})
	Register(&PythonDetector{})
	Register(&GoDetector{})
	Register(&TypeScriptDetector{})
}
