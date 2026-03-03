package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Single(t *testing.T) {
	root := t.TempDir()
	// Single package.json at root → single topology.
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"app"}`), 0644)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologySingle {
		t.Errorf("expected single, got %s", topo)
	}
}

func TestDetect_Monorepo(t *testing.T) {
	root := t.TempDir()
	// Two subdirs with tech files → monorepo.
	os.MkdirAll(filepath.Join(root, "frontend"), 0755)
	os.WriteFile(filepath.Join(root, "frontend", "package.json"), []byte(`{}`), 0644)
	os.MkdirAll(filepath.Join(root, "backend"), 0755)
	os.WriteFile(filepath.Join(root, "backend", "pom.xml"), []byte(`<project/>`), 0644)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMonorepo {
		t.Errorf("expected monorepo, got %s", topo)
	}
}

func TestDetect_Multirepo(t *testing.T) {
	root := t.TempDir()
	// Two subdirs each with .git → multirepo.
	for _, name := range []string{"repo-a", "repo-b"} {
		dir := filepath.Join(root, name)
		os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	}

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMultirepo {
		t.Errorf("expected multirepo, got %s", topo)
	}
}

func TestDetectRepos(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"frontend", "backend", "shared"} {
		os.MkdirAll(filepath.Join(root, name, ".git"), 0755)
	}
	// One non-repo directory.
	os.MkdirAll(filepath.Join(root, "docs"), 0755)

	repos, err := DetectRepos(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 3 {
		t.Errorf("expected 3 repos, got %d", len(repos))
	}
}
