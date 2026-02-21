package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// ErrVersion is returned by Parse when the user requests --version.
var ErrVersion = errors.New("version requested")

// Config holds all runtime settings for the tool.
type Config struct {
	TestPaths []string
	GodotPath string
	Verbose   bool
	Timeout   time.Duration
}

// Parse parses CLI arguments and resolves configuration.
// args should be os.Args[1:] in normal usage.
func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("gdunit4-test-runner", flag.ContinueOnError)

	var godotPath string
	var verbose bool
	var showVersion bool
	var timeout time.Duration

	fs.StringVar(&godotPath, "godot-path", "", "path to Godot binary")
	fs.BoolVar(&verbose, "verbose", false, "stream Godot output to stderr")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.DurationVar(&timeout, "timeout", 0, "kill Godot after this duration (e.g. 30s); 0 means no timeout")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gdunit4-test-runner [options] [paths...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --godot-path <path>  path to Godot binary\n")
		fmt.Fprintf(os.Stderr, "  --verbose            stream Godot output to stderr\n")
		fmt.Fprintf(os.Stderr, "  --timeout <duration> kill Godot after this duration (e.g. 30s); 0 means no timeout\n")
		fmt.Fprintf(os.Stderr, "  --version            print version and exit\n")
		fmt.Fprintf(os.Stderr, "  --help               show this help\n")
		fmt.Fprintf(os.Stderr, "\nIf no paths are given, the current directory is used.\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if showVersion {
		return nil, ErrVersion
	}

	testPaths := fs.Args()
	if len(testPaths) == 0 {
		testPaths = []string{"."}
	}

	resolvedGodot, err := resolveGodotPath(godotPath)
	if err != nil {
		return nil, err
	}

	return &Config{
		TestPaths: testPaths,
		GodotPath: resolvedGodot,
		Verbose:   verbose,
		Timeout:   timeout,
	}, nil
}

// resolveGodotPath resolves the Godot binary path using the priority:
// 1. explicit flag value
// 2. GODOT_PATH environment variable
// 3. "godot" found via PATH lookup
func resolveGodotPath(flagValue string) (string, error) {
	candidates := []string{}
	if flagValue != "" {
		candidates = append(candidates, flagValue)
	}
	if env := os.Getenv("GODOT_PATH"); env != "" {
		candidates = append(candidates, env)
	}

	for _, c := range candidates {
		if isExecutable(c) {
			return c, nil
		}
		return "", fmt.Errorf("Godot binary not found or not executable: %s", c)
	}

	// Fall back to PATH lookup.
	path, err := exec.LookPath("godot")
	if err != nil {
		return "", errors.New("Godot binary not found; set --godot-path or GODOT_PATH")
	}
	return path, nil
}

// isExecutable reports whether path exists and is executable.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
