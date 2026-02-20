package config

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeDummyExecutable creates a dummy executable file in dir and returns its path.
func makeDummyExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to create dummy executable: %v", err)
	}
	return path
}

func TestParse_DefaultToCurrentDir(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	cfg, err := Parse([]string{"--godot-path", godot})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.TestPaths) != 1 || cfg.TestPaths[0] != "." {
		t.Errorf("TestPaths = %v, want [\".\"]", cfg.TestPaths)
	}
}

func TestParse_SinglePath(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	cfg, err := Parse([]string{"--godot-path", godot, "/tmp/tests"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.TestPaths) != 1 || cfg.TestPaths[0] != "/tmp/tests" {
		t.Errorf("TestPaths = %v, want [\"/tmp/tests\"]", cfg.TestPaths)
	}
}

func TestParse_MultiplePaths(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	cfg, err := Parse([]string{"--godot-path", godot, "tests/unit", "tests/integration", "foo.gd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"tests/unit", "tests/integration", "foo.gd"}
	if len(cfg.TestPaths) != len(want) {
		t.Fatalf("TestPaths = %v, want %v", cfg.TestPaths, want)
	}
	for i, p := range want {
		if cfg.TestPaths[i] != p {
			t.Errorf("TestPaths[%d] = %q, want %q", i, cfg.TestPaths[i], p)
		}
	}
}

func TestParse_HelpReturnsErrHelp(t *testing.T) {
	_, err := Parse([]string{"--help"})
	if err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestParse_VersionReturnsErrVersion(t *testing.T) {
	_, err := Parse([]string{"--version"})
	if err != ErrVersion {
		t.Fatalf("expected ErrVersion, got %v", err)
	}
}

func TestParse_UnknownFlag(t *testing.T) {
	_, err := Parse([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestParse_GodotPathFromFlag(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	cfg, err := Parse([]string{"--godot-path", godot, "/tmp/tests"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GodotPath != godot {
		t.Errorf("GodotPath = %q, want %q", cfg.GodotPath, godot)
	}
	if len(cfg.TestPaths) != 1 || cfg.TestPaths[0] != "/tmp/tests" {
		t.Errorf("TestPaths = %v, want [\"/tmp/tests\"]", cfg.TestPaths)
	}
}

func TestParse_GodotPathFromEnv(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	t.Setenv("GODOT_PATH", godot)

	cfg, err := Parse([]string{"/tmp/tests"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GodotPath != godot {
		t.Errorf("GodotPath = %q, want %q", cfg.GodotPath, godot)
	}
}

func TestParse_GodotPathFlagTakesPrecedenceOverEnv(t *testing.T) {
	dir := t.TempDir()
	godotFlag := makeDummyExecutable(t, dir, "godot-flag")
	godotEnv := makeDummyExecutable(t, dir, "godot-env")

	t.Setenv("GODOT_PATH", godotEnv)

	cfg, err := Parse([]string{"--godot-path", godotFlag, "/tmp/tests"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GodotPath != godotFlag {
		t.Errorf("GodotPath = %q, want %q (flag should take precedence)", cfg.GodotPath, godotFlag)
	}
}

func TestParse_VerboseLongFlag(t *testing.T) {
	dir := t.TempDir()
	godot := makeDummyExecutable(t, dir, "godot")

	cfg, err := Parse([]string{"--godot-path", godot, "--verbose"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true when --verbose is set")
	}
}

func TestParse_GodotPathNotExecutable(t *testing.T) {
	dir := t.TempDir()
	// Create a non-executable file
	path := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse([]string{"--godot-path", path, "/tmp/tests"})
	if err == nil {
		t.Fatal("expected error for non-executable godot path, got nil")
	}
}

func TestParse_GodotPathNotFound(t *testing.T) {
	_, err := Parse([]string{"--godot-path", "/nonexistent/godot", "/tmp/tests"})
	if err == nil {
		t.Fatal("expected error for nonexistent godot path, got nil")
	}
}
