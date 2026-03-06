package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextJSDetector(t *testing.T) {
	d := &NextJSDetector{}
	if d.Name() != "nextjs" {
		t.Fatalf("expected name nextjs, got %s", d.Name())
	}

	root := t.TempDir()

	// Not a next.js project — no package.json.
	ok, _ := d.Detect(root)
	if ok {
		t.Error("expected false for empty directory")
	}

	// Write package.json with next dep.
	writeJSON(t, root, `{"dependencies":{"next":"14.0.0","react":"18.0.0"}}`)
	ok, conf := d.Detect(root)
	if !ok {
		t.Error("expected true with next in dependencies")
	}
	if conf < 0.9 {
		t.Errorf("expected confidence >= 0.9, got %.2f", conf)
	}

	// Adding next.config.js should boost to 1.0.
	if err := os.WriteFile(filepath.Join(root, "next.config.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, conf = d.Detect(root)
	if conf != 1.0 {
		t.Errorf("expected 1.0 confidence with next.config.js, got %.2f", conf)
	}
}

func TestReactDetector(t *testing.T) {
	d := &ReactDetector{}

	root := t.TempDir()

	// No package.json → false.
	ok, _ := d.Detect(root)
	if ok {
		t.Error("expected false for empty directory")
	}

	// React without Next.js → high confidence.
	writeJSON(t, root, `{"dependencies":{"react":"18.0.0"},"devDependencies":{"vite":"5.0.0"}}`)
	ok, conf := d.Detect(root)
	if !ok {
		t.Error("expected true for react project")
	}
	if conf < 0.85 {
		t.Errorf("expected conf >= 0.85, got %.2f", conf)
	}

	// React WITH Next.js → lower confidence (nextjs wins).
	writeJSON(t, root, `{"dependencies":{"react":"18.0.0","next":"14.0.0"}}`)
	_, conf = d.Detect(root)
	if conf > 0.6 {
		t.Errorf("expected conf < 0.6 when next is present, got %.2f", conf)
	}
}

func TestDotNetDetector(t *testing.T) {
	d := &DotNetDetector{}

	root := t.TempDir()

	ok, _ := d.Detect(root)
	if ok {
		t.Error("expected false for empty directory")
	}

	// Solution file → 1.0.
	if err := os.WriteFile(filepath.Join(root, "MyApp.sln"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	ok, conf := d.Detect(root)
	if !ok || conf != 1.0 {
		t.Errorf("expected true/1.0 for .sln, got %v/%.2f", ok, conf)
	}
}

func TestAngularDetector(t *testing.T) {
	d := &AngularDetector{}

	root := t.TempDir()
	ok, _ := d.Detect(root)
	if ok {
		t.Error("expected false for empty directory")
	}

	if err := os.WriteFile(filepath.Join(root, "angular.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, conf := d.Detect(root)
	if !ok || conf != 1.0 {
		t.Errorf("expected true/1.0 for angular.json, got %v/%.2f", ok, conf)
	}
}

func TestJavaDetector(t *testing.T) {
	d := &JavaDetector{}

	root := t.TempDir()
	ok, _ := d.Detect(root)
	if ok {
		t.Error("expected false for empty directory")
	}

	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, conf := d.Detect(root)
	if !ok || conf != 1.0 {
		t.Errorf("expected true/1.0 for pom.xml, got %v/%.2f", ok, conf)
	}
}

func TestComposite_MonorepoDetection(t *testing.T) {
	root := t.TempDir()

	// Frontend: React.
	frontendDir := filepath.Join(root, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSONAt(t, frontendDir, `{"dependencies":{"react":"18.0.0"}}`)

	// Backend: .NET.
	backendDir := filepath.Join(root, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "App.sln"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	layers, err := Composite(root)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	if len(layers) < 2 {
		t.Errorf("expected >= 2 layers, got %d: %+v", len(layers), layers)
	}

	stacks := map[string]bool{}
	for _, l := range layers {
		stacks[l.Stack] = true
	}
	if !stacks["react"] {
		t.Error("expected react layer")
	}
	if !stacks["dotnet"] {
		t.Error("expected dotnet layer")
	}
}

func TestComposite_ContainerDirDetection(t *testing.T) {
	root := t.TempDir()

	// apps/frontend with React.
	frontendDir := filepath.Join(root, "apps", "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSONAt(t, frontendDir, `{"dependencies":{"react":"18.0.0"}}`)

	layers, err := Composite(root)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}

	found := false
	for _, l := range layers {
		if l.Stack == "react" && l.Path == "./apps/frontend" {
			found = true
			if l.Name != "frontend" {
				t.Errorf("expected layer name 'frontend', got %q", l.Name)
			}
		}
	}
	if !found {
		t.Errorf("expected react layer at ./apps/frontend, got %+v", layers)
	}
}

func TestComposite_PackagesContainerDir(t *testing.T) {
	root := t.TempDir()

	sharedDir := filepath.Join(root, "packages", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "tsconfig.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	layers, err := Composite(root)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}

	found := false
	for _, l := range layers {
		if l.Stack == "typescript" && l.Path == "./packages/shared" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected typescript layer at ./packages/shared, got %+v", layers)
	}
}

func TestComposite_MixedRootAndContainer(t *testing.T) {
	root := t.TempDir()

	// Root go.mod.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0644); err != nil {
		t.Fatal(err)
	}

	// apps/web with React.
	webDir := filepath.Join(root, "apps", "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSONAt(t, webDir, `{"dependencies":{"react":"18.0.0"}}`)

	layers, err := Composite(root)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}

	stacks := map[string]bool{}
	for _, l := range layers {
		stacks[l.Stack] = true
	}
	if !stacks["go"] {
		t.Error("expected go layer")
	}
	if !stacks["react"] {
		t.Error("expected react layer")
	}
}

func TestComposite_ContainerRootDedup(t *testing.T) {
	root := t.TempDir()

	// apps/ has workspace root package.json + child with package.json.
	appsDir := filepath.Join(root, "apps")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSONAt(t, appsDir, `{"dependencies":{"react":"18.0.0"}}`)

	webDir := filepath.Join(appsDir, "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSONAt(t, webDir, `{"dependencies":{"react":"18.0.0"}}`)

	layers, err := Composite(root)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}

	// The child at ./apps/web should be detected; container-level ./apps
	// should not appear since children were found.
	childFound := false
	for _, l := range layers {
		if l.Path == "./apps/web" && l.Stack == "react" {
			childFound = true
		}
		if l.Path == "./apps" {
			t.Errorf("expected no layer at ./apps when children exist, but found: %+v", l)
		}
	}
	if !childFound {
		t.Errorf("expected react layer at ./apps/web, got %+v", layers)
	}
}

// helpers

func writeJSON(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONAt(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
