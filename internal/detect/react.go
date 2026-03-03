package detect

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
)

// ReactDetector identifies a React (non-Next.js) project.
type ReactDetector struct{}

func (d *ReactDetector) Name() string { return "react" }

func (d *ReactDetector) Detect(root string) (bool, float64) {
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return false, 0
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, 0
	}

	// React must be present.
	_, hasReact := pkg.Dependencies["react"]
	if !hasReact {
		_, hasReact = pkg.DevDependencies["react"]
	}
	if !hasReact {
		return false, 0
	}

	// If Next.js is also present, let NextJSDetector take precedence with a
	// lower confidence so the composite builder can pick it up.
	_, hasNext := pkg.Dependencies["next"]
	if !hasNext {
		_, hasNext = pkg.DevDependencies["next"]
	}
	if hasNext {
		// Still report as react but with lower confidence so nextjs wins.
		return true, 0.55
	}

	conf := 0.85

	// Boost for Vite or Create React App signals.
	_, hasVite := pkg.DevDependencies["vite"]
	if hasVite {
		conf = 0.95
	}
	if fileExists(root, "vite.config.ts") || fileExists(root, "vite.config.js") {
		conf = 1.0
	}

	return true, conf
}

func (d *ReactDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["react"]
}
