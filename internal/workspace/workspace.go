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

		// Another git repo in the same parent?
		if isGitRepo(subAbs) {
			subRepoCount++
		}

		// Does this subdirectory look like a project layer?
		if looksLikeProjectDir(subAbs) {
			techDirCount++
		}
	}

	switch {
	case subRepoCount >= 2:
		// Parent dir of multiple repos → multi-repo workspace.
		return TopologyMultirepo, nil
	case techDirCount >= 2:
		// One repo with multiple tech subdirectories → monorepo.
		return TopologyMonorepo, nil
	default:
		return TopologySingle, nil
	}
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
