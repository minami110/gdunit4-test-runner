package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildArgs_SinglePath(t *testing.T) {
	resPath := "res://tests/unit"
	args := BuildArgs([]string{resPath})

	// Must include --headless
	if !contains(args, "--headless") {
		t.Error("args should contain --headless")
	}
	// Must include -s and -d
	if !contains(args, "-s") {
		t.Error("args should contain -s")
	}
	if !contains(args, "-d") {
		t.Error("args should contain -d")
	}
	// Must include the GdUnitCmdTool.gd script
	if !contains(args, "res://addons/gdUnit4/bin/GdUnitCmdTool.gd") {
		t.Error("args should contain the GdUnitCmdTool.gd path")
	}
	// Must include -a followed by resPath
	idx := indexOf(args, "-a")
	if idx == -1 || idx+1 >= len(args) {
		t.Fatal("args should contain -a <resPath>")
	}
	if args[idx+1] != resPath {
		t.Errorf("arg after -a = %q, want %q", args[idx+1], resPath)
	}
	// Must include --ignoreHeadlessMode
	if !contains(args, "--ignoreHeadlessMode") {
		t.Error("args should contain --ignoreHeadlessMode")
	}
	// Must include -c
	if !contains(args, "-c") {
		t.Error("args should contain -c")
	}
}

func TestBuildArgs_MultiplePaths(t *testing.T) {
	resPaths := []string{"res://tests/unit", "res://tests/integration"}
	args := BuildArgs(resPaths)

	// Count -a occurrences.
	count := 0
	for _, a := range args {
		if a == "-a" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 -a flags, got %d", count)
	}

	// Both paths must appear.
	if !contains(args, "res://tests/unit") {
		t.Error("args should contain res://tests/unit")
	}
	if !contains(args, "res://tests/integration") {
		t.Error("args should contain res://tests/integration")
	}

	// Verify ordering: -a path1 -a path2
	idx1 := indexOf(args, "-a")
	if idx1 == -1 || idx1+1 >= len(args) || args[idx1+1] != "res://tests/unit" {
		t.Errorf("first -a should be followed by res://tests/unit, args = %v", args)
	}
	idx2 := indexOf(args[idx1+2:], "-a")
	if idx2 == -1 {
		t.Fatal("expected second -a flag")
	}
	idx2 += idx1 + 2
	if idx2+1 >= len(args) || args[idx2+1] != "res://tests/integration" {
		t.Errorf("second -a should be followed by res://tests/integration, args = %v", args)
	}
}

func TestRun_CapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "fake-godot.sh")
	// Write a fake godot script that prints to stdout and exits 0
	content := "#!/bin/sh\necho 'hello from godot'\necho 'error line' >&2\nexit 0\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Run(script, dir, []string{"res://tests"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(result.LogFile)

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	data, err := os.ReadFile(result.LogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello from godot") {
		t.Errorf("log file should contain 'hello from godot', got: %s", string(data))
	}
}

func TestRun_NonZeroExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "fake-godot-fail.sh")
	content := "#!/bin/sh\necho 'test failed'\nexit 100\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Run(script, dir, []string{"res://tests"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(result.LogFile)

	if result.ExitCode != 100 {
		t.Errorf("ExitCode = %d, want 100", result.ExitCode)
	}
}

func TestRun_LogFileExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "fake-godot-noop.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Run(script, dir, []string{"res://tests"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(result.LogFile)

	if result.LogFile == "" {
		t.Error("LogFile should not be empty")
	}
	if _, err := os.Stat(result.LogFile); err != nil {
		t.Errorf("log file should exist: %v", err)
	}
}

func TestRun_BinaryNotFound(t *testing.T) {
	_, err := Run("/nonexistent/godot", "/tmp", []string{"res://tests"}, false)
	if err == nil {
		t.Fatal("expected error when godot binary not found, got nil")
	}
}

// contains reports whether slice contains elem.
func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if s == elem {
			return true
		}
	}
	return false
}

// indexOf returns the index of elem in slice, or -1 if not found.
func indexOf(slice []string, elem string) int {
	for i, s := range slice {
		if s == elem {
			return i
		}
	}
	return -1
}
