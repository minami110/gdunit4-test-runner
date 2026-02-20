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

	result, err := Detect(testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPath != "res://tests/unit" {
		t.Errorf("ResPath = %q, want %q", result.ResPath, "res://tests/unit")
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

	result, err := Detect(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPath != "res://tests/MyTest.gd" {
		t.Errorf("ResPath = %q, want %q", result.ResPath, "res://tests/MyTest.gd")
	}
}

func TestDetect_ProjectRootItself(t *testing.T) {
	root := makeProject(t)

	result, err := Detect(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPath != "res://." {
		t.Errorf("ResPath = %q, want res://.", result.ResPath)
	}
}

func TestDetect_NoProjectGodot(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect(dir)
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

	_, err := Detect(root)
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

	result, err := Detect(deep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProjectDir != root {
		t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, root)
	}
	if result.ResPath != "res://a/b/c/d" {
		t.Errorf("ResPath = %q, want res://a/b/c/d", result.ResPath)
	}
}
