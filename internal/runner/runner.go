package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// RunResult holds the outcome of running Godot.
type RunResult struct {
	ExitCode int
	LogFile  string // caller is responsible for removing this file
}

// BuildArgs constructs the Godot command arguments for gdUnit4.
// Each path in resPaths is passed as a separate -a flag.
func BuildArgs(resPaths []string) []string {
	args := []string{
		"--headless",
		"-s", "-d",
		"res://addons/gdUnit4/bin/GdUnitCmdTool.gd",
	}
	for _, p := range resPaths {
		args = append(args, "-a", p)
	}
	args = append(args, "--ignoreHeadlessMode", "-c")
	return args
}

// Run executes Godot with gdUnit4 arguments from projectDir.
// Output is captured to a temporary log file; if verbose is true it is also written to stderr.
// If timeout > 0, the process is killed after that duration.
func Run(godotPath, projectDir string, resPaths []string, verbose bool, timeout time.Duration) (*RunResult, error) {
	args := BuildArgs(resPaths)

	var cmd *exec.Cmd
	var cancelCtx context.CancelFunc
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cancelCtx = cancel
		cmd = exec.CommandContext(ctx, godotPath, args...)
	} else {
		cmd = exec.Command(godotPath, args...)
	}
	cmd.Dir = projectDir

	tmpFile, err := os.CreateTemp("", "gdunit4-runner-*.log")
	if err != nil {
		if cancelCtx != nil {
			cancelCtx()
		}
		return nil, fmt.Errorf("failed to create temp log file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Always pass *os.File directly — avoids pipe creation that hangs on Windows
	// when child processes inherit the pipe handle and keep it open after Godot exits.
	cmd.Stdout = tmpFile
	cmd.Stderr = tmpFile

	// Create a pipe for stdin and close the write end immediately.
	// This ensures the child process receives a proper EOF that sets feof(stdin)=true.
	// Godot's LocalDebugger checks feof(stdin) on empty input — if false,
	// it loops the debug> prompt indefinitely. NUL (os.DevNull on Windows)
	// may not set feof, but a closed pipe always does.
	pr, pw, pipeErr := os.Pipe()
	if pipeErr != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		if cancelCtx != nil {
			cancelCtx()
		}
		return nil, fmt.Errorf("failed to create stdin pipe: %w", pipeErr)
	}
	pw.Close()
	cmd.Stdin = pr
	defer pr.Close()

	var wg sync.WaitGroup
	var stopTail chan struct{}
	if verbose {
		stopTail = make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			tailToStderr(tmpPath, stopTail)
		}()
	}

	runErr := cmd.Run()

	if cancelCtx != nil {
		cancelCtx()
	}

	// Close the temp file before returning so callers can read it.
	if closeErr := tmpFile.Close(); closeErr != nil && runErr == nil {
		runErr = closeErr
	}

	if verbose {
		close(stopTail)
		wg.Wait()
	}

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if timeout > 0 && runErr == context.DeadlineExceeded {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("Godot process timed out after %s", timeout)
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

// tailToStderr reads path and writes new data to stderr until stop is closed,
// then drains any remaining data and returns.
func tailToStderr(path string, stop <-chan struct{}) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			os.Stderr.Write(buf[:n])
		}
		if err != nil {
			select {
			case <-stop:
				// Process exited — drain remaining data and return.
				io.Copy(os.Stderr, f)
				return
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}
