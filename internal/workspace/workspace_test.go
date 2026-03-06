package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Single(t *testing.T) {
	root := t.TempDir()
	// Single package.json at root → single topology.
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err := os.MkdirAll(filepath.Join(root, "frontend"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "frontend", "package.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "pom.xml"), []byte(`<project/>`), 0644); err != nil {
		t.Fatal(err)
	}

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
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMultirepo {
		t.Errorf("expected multirepo, got %s", topo)
	}
}

func TestDetect_ContainerDirMonorepo(t *testing.T) {
	root := t.TempDir()
	// apps/frontend/package.json + apps/backend/go.mod → monorepo.
	mkdirWrite(t, root, "apps/frontend", "package.json", `{}`)
	mkdirWrite(t, root, "apps/backend", "go.mod", `module example`)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMonorepo {
		t.Errorf("expected monorepo, got %s", topo)
	}
}

func TestDetect_EmptyContainerDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps"), 0755); err != nil {
		t.Fatal(err)
	}

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologySingle {
		t.Errorf("expected single, got %s", topo)
	}
}

func TestDetect_ContainerAsProject(t *testing.T) {
	root := t.TempDir()
	// apps/ has package.json but no project children → single (1 tech dir).
	mkdirWrite(t, root, "apps", "package.json", `{}`)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologySingle {
		t.Errorf("expected single, got %s", topo)
	}
}

func TestDetect_ContainerWithTechMarkersAndChildren(t *testing.T) {
	root := t.TempDir()
	// apps/ has its own package.json (workspace root) AND project children → monorepo.
	mkdirWrite(t, root, "apps", "package.json", `{}`)
	mkdirWrite(t, root, "apps/web", "package.json", `{}`)
	mkdirWrite(t, root, "apps/api", "go.mod", `module api`)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMonorepo {
		t.Errorf("expected monorepo, got %s", topo)
	}
}

func TestDetect_MonorepoSignalTiebreaker(t *testing.T) {
	root := t.TempDir()
	// turbo.json at root + single project in apps/ → monorepo via signal.
	if err := os.WriteFile(filepath.Join(root, "turbo.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	mkdirWrite(t, root, "apps/web", "package.json", `{}`)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMonorepo {
		t.Errorf("expected monorepo, got %s", topo)
	}
}

func TestDetect_FlatMonorepoRegression(t *testing.T) {
	root := t.TempDir()
	mkdirWrite(t, root, "frontend", "package.json", `{}`)
	mkdirWrite(t, root, "backend", "go.mod", `module backend`)

	topo, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if topo != TopologyMonorepo {
		t.Errorf("expected monorepo, got %s", topo)
	}
}

func mkdirWrite(t *testing.T, root, dir, file, content string) {
	t.Helper()
	abs := filepath.Join(root, dir)
	if err := os.MkdirAll(abs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abs, file), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectRepos(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"frontend", "backend", "shared"} {
		if err := os.MkdirAll(filepath.Join(root, name, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// One non-repo directory.
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}

	repos, err := DetectRepos(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 3 {
		t.Errorf("expected 3 repos, got %d", len(repos))
	}
}
