package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// Topology represents how repos are organised.
type Topology string

const (
	TopologyMonorepo  Topology = "monorepo"
	TopologyMultirepo Topology = "multirepo"
	TopologySingle    Topology = "single"
)

// RepoInfo is a detected repository in a workspace.
type RepoInfo struct {
	Name string
	Path string // Absolute path
}

var knownContainerDirs = map[string]bool{
	"apps": true, "packages": true, "libs": true,
	"libraries": true, "services": true, "projects": true,
}

var monorepoSignalFiles = []string{
	"turbo.json", "nx.json", "pnpm-workspace.yaml", "lerna.json",
}

// Detect determines the workspace topology from root.
func Detect(root string) (Topology, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return TopologySingle, err
	}

	subRepoCount := 0
	techDirCount := 0

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		subAbs := filepath.Join(root, e.Name())

		if isGitRepo(subAbs) {
			subRepoCount++
		}

		if knownContainerDirs[e.Name()] {
			techDirCount += countContainerProjects(subAbs)
			continue
		}

		if looksLikeProjectDir(subAbs) {
			techDirCount++
		}
	}

	hasSignal := hasMonorepoSignal(root)

	switch {
	case subRepoCount >= 2:
		return TopologyMultirepo, nil
	case techDirCount >= 2:
		return TopologyMonorepo, nil
	case techDirCount >= 1 && hasSignal:
		return TopologyMonorepo, nil
	default:
		return TopologySingle, nil
	}
}

// countContainerProjects counts project-like children inside a container dir.
// If no children look like projects but the container itself does, it counts as 1.
func countContainerProjects(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if looksLikeProjectDir(filepath.Join(dir, e.Name())) {
			count++
		}
	}
	if count > 0 {
		return count
	}
	if looksLikeProjectDir(dir) {
		return 1
	}
	return 0
}

func hasMonorepoSignal(root string) bool {
	for _, f := range monorepoSignalFiles {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

// DetectRepos finds sibling git repositories relative to root.
func DetectRepos(root string) ([]RepoInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var repos []RepoInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		abs := filepath.Join(root, e.Name())
		if isGitRepo(abs) {
			repos = append(repos, RepoInfo{Name: e.Name(), Path: abs})
		}
	}
	return repos, nil
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func looksLikeProjectDir(dir string) bool {
	// Has a package.json, *.csproj, pom.xml, go.mod, etc.
	signals := []string{
		"package.json", "go.mod", "pom.xml", "build.gradle",
		"requirements.txt", "pyproject.toml", "angular.json",
		"tsconfig.json",
	}
	for _, s := range signals {
		if _, err := os.Stat(filepath.Join(dir, s)); err == nil {
			return true
		}
	}
	// Or has *.csproj.
	matches, _ := filepath.Glob(filepath.Join(dir, "*.csproj"))
	return len(matches) > 0
}
