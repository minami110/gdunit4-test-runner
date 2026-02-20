package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

// Config holds all runtime settings for the tool.
type Config struct {
	TestPath  string
	GodotPath string
	Verbose   bool
}

// Parse parses CLI arguments and resolves configuration.
// args should be os.Args[1:] in normal usage.
func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("gdunit4-test-runner", flag.ContinueOnError)

	var testPath string
	var godotPath string
	var verbose bool

	fs.StringVar(&testPath, "path", "", "path to test directory or file (required)")
	fs.StringVar(&godotPath, "godot-path", "", "path to Godot binary")
	fs.BoolVar(&verbose, "v", false, "stream Godot output to stderr")
	fs.BoolVar(&verbose, "verbose", false, "stream Godot output to stderr")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gdunit4-test-runner [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --path <path>        path to test directory or file (required)\n")
		fmt.Fprintf(os.Stderr, "  --godot-path <path>  path to Godot binary\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose        stream Godot output to stderr\n")
		fmt.Fprintf(os.Stderr, "  -h, --help           show this help\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if testPath == "" {
		return nil, errors.New("--path is required")
	}

	resolvedGodot, err := resolveGodotPath(godotPath)
	if err != nil {
		return nil, err
	}

	return &Config{
		TestPath:  testPath,
		GodotPath: resolvedGodot,
		Verbose:   verbose,
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
	return info.Mode()&0o111 != 0
}
