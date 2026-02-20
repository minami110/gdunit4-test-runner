package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunResult holds the outcome of running Godot.
type RunResult struct {
	ExitCode int
	LogFile  string // caller is responsible for removing this file
}

// BuildArgs constructs the Godot command arguments for gdUnit4.
func BuildArgs(resPath string) []string {
	return []string{
		"--headless",
		"-s", "-d",
		"res://addons/gdUnit4/bin/GdUnitCmdTool.gd",
		"-a", resPath,
		"--ignoreHeadlessMode",
		"-c",
	}
}

// Run executes Godot with gdUnit4 arguments from projectDir.
// Output is captured to a temporary log file; if verbose is true it is also written to stderr.
func Run(godotPath, projectDir, resPath string, verbose bool) (*RunResult, error) {
	args := BuildArgs(resPath)
	cmd := exec.Command(godotPath, args...)
	cmd.Dir = projectDir

	tmpFile, err := os.CreateTemp("", "gdunit4-runner-*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp log file: %w", err)
	}
	tmpPath := tmpFile.Name()

	var writer io.Writer
	if verbose {
		writer = io.MultiWriter(tmpFile, os.Stderr)
	} else {
		writer = tmpFile
	}

	cmd.Stdout = writer
	cmd.Stderr = writer

	runErr := cmd.Run()
	// Close the temp file before returning so callers can read it.
	if closeErr := tmpFile.Close(); closeErr != nil && runErr == nil {
		runErr = closeErr
	}

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g. binary not found at exec time).
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to run Godot: %w", runErr)
		}
	}

	return &RunResult{
		ExitCode: exitCode,
		LogFile:  tmpPath,
	}, nil
}
