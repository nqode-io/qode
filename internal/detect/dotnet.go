package detect

import (
	"github.com/nqode/qode/internal/config"
)

// DotNetDetector identifies a .NET project.
type DotNetDetector struct{}

func (d *DotNetDetector) Name() string { return "dotnet" }

func (d *DotNetDetector) Detect(root string) (bool, float64) {
	// High confidence: solution file.
	if globExists(root, "*.sln") {
		return true, 1.0
	}
	// Good confidence: project file.
	if globExists(root, "*.csproj") || globExists(root, "*.fsproj") || globExists(root, "*.vbproj") {
		return true, 0.95
	}
	// Lower confidence: global.json.
	if fileExists(root, "global.json") {
		return true, 0.7
	}
	return false, 0
}

func (d *DotNetDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["dotnet"]
}
