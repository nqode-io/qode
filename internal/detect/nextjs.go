package detect

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
)

// NextJSDetector identifies a Next.js project.
type NextJSDetector struct{}

func (d *NextJSDetector) Name() string { return "nextjs" }

func (d *NextJSDetector) Detect(root string) (bool, float64) {
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

	conf := 0.0
	if _, ok := pkg.Dependencies["next"]; ok {
		conf = 0.95
	} else if _, ok := pkg.DevDependencies["next"]; ok {
		conf = 0.85
	}

	if conf == 0 {
		return false, 0
	}

	// Boost confidence if next.config.js/ts exists.
	if fileExists(root, "next.config.js") || fileExists(root, "next.config.ts") || fileExists(root, "next.config.mjs") {
		conf = 1.0
	}

	return true, conf
}

func (d *NextJSDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["nextjs"]
}
