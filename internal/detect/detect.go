package detect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const minConfidence = 0.5

var knownContainerDirs = map[string]bool{
	"apps": true, "packages": true, "libs": true,
	"libraries": true, "services": true, "projects": true,
}

var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	".git": true, "__pycache__": true, "bin": true, "obj": true,
}

// Composite scans root and every immediate subdirectory, returning all
// detected technology layers sorted by path then confidence.
func Composite(root string) ([]DetectedLayer, error) {
	var layers []DetectedLayer

	// Check root itself.
	rootLayers := detectAt(root, ".", 0)
	layers = append(layers, rootLayers...)

	// Check immediate subdirectories.
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if skipDirs[e.Name()] {
			continue
		}
		subPath := e.Name()
		subAbs := filepath.Join(root, subPath)

		// Recurse into known container directories.
		if knownContainerDirs[subPath] {
			childLayers := detectContainerChildren(subAbs, subPath)
			if len(childLayers) > 0 {
				layers = append(layers, childLayers...)
			} else {
				layers = append(layers, detectAt(subAbs, "./"+subPath, 1)...)
			}
			continue
		}

		subLayers := detectAt(subAbs, "./"+subPath, 1)
		layers = append(layers, subLayers...)
	}

	// Deduplicate: if root has a stack that is also found in a subdirectory
	// (e.g. a Next.js monorepo where root has package.json but ./frontend also
	// does), prefer the subdirectory entry if it has higher confidence.
	layers = dedup(layers)

	sort.Slice(layers, func(i, j int) bool {
		if layers[i].Path != layers[j].Path {
			return layers[i].Path < layers[j].Path
		}
		return layers[i].Confidence > layers[j].Confidence
	})

	return layers, nil
}

// detectContainerChildren scans children of a container directory (e.g. apps/)
// and runs detection on each child that looks like a project.
func detectContainerChildren(containerAbs, containerName string) []DetectedLayer {
	entries, err := os.ReadDir(containerAbs)
	if err != nil {
		return nil
	}
	var layers []DetectedLayer
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if skipDirs[e.Name()] {
			continue
		}
		childAbs := filepath.Join(containerAbs, e.Name())
		childRel := "./" + containerName + "/" + e.Name()
		layers = append(layers, detectAt(childAbs, childRel, 2)...)
	}
	return layers
}

// detectAt runs all registered detectors against dir and returns layers that
// meet the minimum confidence threshold.
func detectAt(dir, relPath string, depth int) []DetectedLayer {
	var found []DetectedLayer
	for _, d := range All() {
		ok, conf := d.Detect(dir)
		if !ok || conf < minConfidence {
			continue
		}
		name := suggestName(d.Name(), relPath, depth)
		found = append(found, DetectedLayer{
			Name:       name,
			Path:       relPath,
			Stack:      d.Name(),
			Confidence: conf,
			Test:       d.DefaultConfig(),
		})
	}
	return found
}

// suggestName creates a human-friendly layer name from the stack and path.
func suggestName(stack, path string, depth int) string {
	if depth == 0 {
		return "default"
	}
	// Use the directory name as the layer name.
	base := filepath.Base(path)
	if base == "." || base == "" {
		return stack
	}
	return base
}

// stackSupersedes maps a higher-specificity stack to the stacks it replaces
// when both are detected at the same path.
var stackSupersedes = map[string][]string{
	"nextjs":  {"react", "typescript"},
	"angular": {"typescript"},
}

// dedup removes layers at the root that have the same stack as a subdirectory
// entry, and removes lower-specificity stacks superseded at the same path.
func dedup(layers []DetectedLayer) []DetectedLayer {
	// Build set of stacks found in subdirectories (for root dedup).
	subStacks := map[string]bool{}
	for _, l := range layers {
		if l.Path != "." {
			subStacks[l.Stack] = true
		}
	}

	// Build per-path set of detected stacks for supersede checks.
	pathStacks := map[string]map[string]bool{}
	for _, l := range layers {
		if pathStacks[l.Path] == nil {
			pathStacks[l.Path] = map[string]bool{}
		}
		pathStacks[l.Path][l.Stack] = true
	}

	result := layers[:0]
	for _, l := range layers {
		// Skip root entry if same stack exists in a sub-path.
		if l.Path == "." && subStacks[l.Stack] {
			continue
		}
		// Skip if a more specific stack at the same path supersedes this one.
		superseded := false
		for dominant, dominated := range stackSupersedes {
			if pathStacks[l.Path][dominant] {
				for _, d := range dominated {
					if l.Stack == d {
						superseded = true
						break
					}
				}
			}
			if superseded {
				break
			}
		}
		if !superseded {
			result = append(result, l)
		}
	}
	return result
}

// fileExists returns true if path exists in dir.
func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// globExists returns true if at least one file matching the pattern exists.
func globExists(dir, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	return err == nil && len(matches) > 0
}
