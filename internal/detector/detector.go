package detector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Result holds the outcome of project detection.
type Result struct {
	ProjectDir string // absolute path to the directory containing project.godot
	ResPath    string // res://-relative path for the test target
}

// Detect finds the Godot project root for testPath and converts testPath to a res:// path.
// It walks up from testPath looking for project.godot, then verifies addons/gdUnit4/ exists.
func Detect(testPath string) (*Result, error) {
	absPath, err := filepath.Abs(testPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	projectDir, err := findProjectRoot(absPath)
	if err != nil {
		return nil, err
	}

	if err := verifyGdUnit4(projectDir); err != nil {
		return nil, err
	}

	resPath, err := toResPath(projectDir, absPath)
	if err != nil {
		return nil, err
	}

	return &Result{
		ProjectDir: projectDir,
		ResPath:    resPath,
	}, nil
}

// findProjectRoot walks up from startPath looking for a directory containing project.godot.
func findProjectRoot(startPath string) (string, error) {
	// Start from startPath itself; if it's a file, start from its directory.
	dir := startPath
	info, err := os.Stat(startPath)
	if err != nil {
		return "", fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		dir = filepath.Dir(startPath)
	}

	for {
		candidate := filepath.Join(dir, "project.godot")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			break
		}
		dir = parent
	}

	return "", errors.New("project.godot not found; point --path to a subdirectory of your Godot project")
}

// verifyGdUnit4 checks that addons/gdUnit4/ exists under projectDir.
func verifyGdUnit4(projectDir string) error {
	addonPath := filepath.Join(projectDir, "addons", "gdUnit4")
	info, err := os.Stat(addonPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("addons/gdUnit4/ not found under %s", projectDir)
	}
	return nil
}

// toResPath converts an absolute testPath to a res://-relative path.
func toResPath(projectDir, testPath string) (string, error) {
	rel, err := filepath.Rel(projectDir, testPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute res:// path: %w", err)
	}
	return "res://" + filepath.ToSlash(rel), nil
}
