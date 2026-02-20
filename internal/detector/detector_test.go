package detector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeProject creates a minimal Godot project structure in a temp dir and returns its root.
func makeProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte("[application]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	addonDir := filepath.Join(root, "addons", "gdUnit4")
	if err := os.MkdirAll(addonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestDetect_DirectoryUnderProject(t *testing.T) {
	root := makeProject(t)
	testsDir := filepath.Join(root, "tests", "unit")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Detect([]string{testsDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPaths[0] != "res://tests/unit" {
		t.Errorf("ResPaths[0] = %q, want %q", result.ResPaths[0], "res://tests/unit")
	}
}

func TestDetect_FileUnderProject(t *testing.T) {
	root := makeProject(t)
	testsDir := filepath.Join(root, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(testsDir, "MyTest.gd")
	if err := os.WriteFile(testFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Detect([]string{testFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPaths[0] != "res://tests/MyTest.gd" {
		t.Errorf("ResPaths[0] = %q, want %q", result.ResPaths[0], "res://tests/MyTest.gd")
	}
}

func TestDetect_ProjectRootItself(t *testing.T) {
	root := makeProject(t)

	result, err := Detect([]string{root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPaths[0] != "res://." {
		t.Errorf("ResPaths[0] = %q, want res://.", result.ResPaths[0])
	}
}

func TestDetect_NoProjectGodot(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect([]string{dir})
	if err == nil {
		t.Fatal("expected error when project.godot is missing, got nil")
	}
	if !strings.Contains(err.Error(), "project.godot") {
		t.Errorf("error message should mention project.godot, got: %v", err)
	}
}

func TestDetect_MissingGdUnit4Addon(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte("[application]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Do NOT create addons/gdUnit4

	_, err := Detect([]string{root})
	if err == nil {
		t.Fatal("expected error when addons/gdUnit4 is missing, got nil")
	}
	if !strings.Contains(err.Error(), "addons/gdUnit4") {
		t.Errorf("error message should mention addons/gdUnit4, got: %v", err)
	}
}

func TestDetect_DeepNestedPath(t *testing.T) {
	root := makeProject(t)
	deep := filepath.Join(root, "a", "b", "c", "d")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Detect([]string{deep})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPaths[0] != "res://a/b/c/d" {
		t.Errorf("ResPaths[0] = %q, want res://a/b/c/d", result.ResPaths[0])
	}
}

func TestDetect_MultiplePaths(t *testing.T) {
	root := makeProject(t)
	dir1 := filepath.Join(root, "tests", "unit")
	dir2 := filepath.Join(root, "tests", "integration")
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Detect([]string{dir1, dir2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if len(result.ResPaths) != 2 {
		t.Fatalf("len(ResPaths) = %d, want 2", len(result.ResPaths))
	}
	if result.ResPaths[0] != "res://tests/unit" {
		t.Errorf("ResPaths[0] = %q, want res://tests/unit", result.ResPaths[0])
	}
	if result.ResPaths[1] != "res://tests/integration" {
		t.Errorf("ResPaths[1] = %q, want res://tests/integration", result.ResPaths[1])
	}
}

func TestDetect_CrossProjectError(t *testing.T) {
	// Create two separate Godot projects.
	root1 := makeProject(t)
	root2 := makeProject(t)

	dir1 := filepath.Join(root1, "tests")
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatal(err)
	}
	dir2 := filepath.Join(root2, "tests")
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Detect([]string{dir1, dir2})
	if err == nil {
		t.Fatal("expected error when paths belong to different projects, got nil")
	}
	if !strings.Contains(err.Error(), "different Godot project") {
		t.Errorf("error message should mention different project, got: %v", err)
	}
}
