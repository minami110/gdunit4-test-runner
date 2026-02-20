package detector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Result holds the outcome of project detection.
type Result struct {
	ProjectDir string   // absolute path to the directory containing project.godot
	ResPaths   []string // res://-relative paths for the test targets
}

// Detect finds the Godot project root for testPaths and converts each path to a res:// path.
// It walks up from the first path looking for project.godot, then verifies addons/gdUnit4/ exists.
// All paths must belong to the same Godot project.
func Detect(testPaths []string) (*Result, error) {
	if len(testPaths) == 0 {
		return nil, errors.New("no test paths provided")
	}

	// Use the first path to determine project root.
	firstAbs, err := filepath.Abs(testPaths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	projectDir, err := findProjectRoot(firstAbs)
	if err != nil {
		return nil, err
	}

	if err := verifyGdUnit4(projectDir); err != nil {
		return nil, err
	}

	resPaths := make([]string, 0, len(testPaths))
	for _, p := range testPaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", p, err)
		}

		// Verify this path belongs to the same project by finding its root.
		root, err := findProjectRoot(absPath)
		if err != nil {
			return nil, fmt.Errorf("path %s: %w", p, err)
		}
		if root != projectDir {
			return nil, fmt.Errorf("path %s belongs to a different Godot project (%s), expected %s", p, root, projectDir)
		}

		resPath, err := toResPath(projectDir, absPath)
		if err != nil {
			return nil, err
		}
		resPaths = append(resPaths, resPath)
	}

	return &Result{
		ProjectDir: projectDir,
		ResPaths:   resPaths,
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

	return "", errors.New("project.godot not found; point the path to a subdirectory of your Godot project")
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
