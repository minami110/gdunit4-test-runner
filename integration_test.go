package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// integrationOutput mirrors the JSON structure produced by gdunit4-test-runner.
type integrationOutput struct {
	Summary struct {
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Crashed bool   `json:"crashed"`
		Status  string `json:"status"`
	} `json:"summary"`
	CrashDetails *struct {
		CrashInfo    string `json:"crash_info,omitempty"`
		ScriptErrors string `json:"script_errors,omitempty"`
		EngineErrors string `json:"engine_errors,omitempty"`
	} `json:"crash_details,omitempty"`
	Failures []struct {
		Class    string `json:"class"`
		Method   string `json:"method"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
		Message  string `json:"message"`
	} `json:"failures"`
}

// godotProjectDir returns the absolute path to testdata/godot-project.
func godotProjectDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "godot-project")
}

// skipIfNotReady skips the test if GODOT_PATH is not set or gdUnit4 addon is missing.
func skipIfNotReady(t *testing.T) (godotPath, projectDir string) {
	t.Helper()

	godotPath = os.Getenv("GODOT_PATH")
	if godotPath == "" {
		t.Skip("GODOT_PATH not set; skipping integration tests")
	}

	projectDir = godotProjectDir(t)
	addonDir := filepath.Join(projectDir, "addons", "gdUnit4")
	if _, err := os.Stat(addonDir); os.IsNotExist(err) {
		t.Skipf("gdUnit4 addon not installed at %s; skipping integration tests", addonDir)
	}

	return godotPath, projectDir
}

// buildBinary compiles the gdunit4-test-runner binary into a temp directory.
// Returns the path to the compiled binary.
func buildBinary(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(thisFile)

	binName := "gdunit4-test-runner"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/gdunit4-test-runner")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return binPath
}

// runBinary executes the gdunit4-test-runner binary with the given --path and returns
// the parsed JSON output and the process exit code.
func runBinary(t *testing.T, binPath, godotPath, testPath string) (*integrationOutput, int) {
	t.Helper()

	cmd := exec.Command(binPath, "--path", testPath, "--godot-path", godotPath)
	stdout, err := cmd.Output()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running binary: %v", err)
		}
	}

	var out integrationOutput
	if len(stdout) > 0 {
		if jsonErr := json.Unmarshal(stdout, &out); jsonErr != nil {
			t.Fatalf("failed to parse JSON output: %v\nraw stdout: %s", jsonErr, stdout)
		}
	}

	return &out, exitCode
}

// cleanReports removes the reports/ directory under the godot project to ensure
// each test run starts with a clean state.
func cleanReports(t *testing.T, projectDir string) {
	t.Helper()
	reportsDir := filepath.Join(projectDir, "reports")
	if err := os.RemoveAll(reportsDir); err != nil {
		t.Logf("warning: failed to remove reports dir: %v", err)
	}
}

func TestIntegration_PassingDir(t *testing.T) {
	godotPath, projectDir := skipIfNotReady(t)
	cleanReports(t, projectDir)

	binPath := buildBinary(t)
	testPath := filepath.Join(projectDir, "tests", "passing")

	out, exitCode := runBinary(t, binPath, godotPath, testPath)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if out.Summary.Status != "passed" {
		t.Errorf("expected status 'passed', got %q", out.Summary.Status)
	}
	if out.Summary.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", out.Summary.Failed)
	}
}

func TestIntegration_PassingSingleFile(t *testing.T) {
	godotPath, projectDir := skipIfNotReady(t)
	cleanReports(t, projectDir)

	binPath := buildBinary(t)
	testPath := filepath.Join(projectDir, "tests", "passing", "test_basic_math.gd")

	out, exitCode := runBinary(t, binPath, godotPath, testPath)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if out.Summary.Status != "passed" {
		t.Errorf("expected status 'passed', got %q", out.Summary.Status)
	}
	if out.Summary.Total < 1 {
		t.Errorf("expected total >= 1, got %d", out.Summary.Total)
	}
}

func TestIntegration_Failing(t *testing.T) {
	godotPath, projectDir := skipIfNotReady(t)
	cleanReports(t, projectDir)

	binPath := buildBinary(t)
	testPath := filepath.Join(projectDir, "tests", "failing")

	out, exitCode := runBinary(t, binPath, godotPath, testPath)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if out.Summary.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", out.Summary.Status)
	}
	if len(out.Failures) == 0 {
		t.Error("expected at least one failure entry, got none")
	}
}

func TestIntegration_CompileError(t *testing.T) {
	godotPath, projectDir := skipIfNotReady(t)
	cleanReports(t, projectDir)

	binPath := buildBinary(t)
	testPath := filepath.Join(projectDir, "tests", "compile_error")

	out, exitCode := runBinary(t, binPath, godotPath, testPath)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if out.Summary.Status != "crashed" {
		t.Errorf("expected status 'crashed', got %q", out.Summary.Status)
	}
	if out.CrashDetails == nil {
		t.Error("expected crash_details to be non-nil")
	}
}

func TestIntegration_NotTests(t *testing.T) {
	godotPath, projectDir := skipIfNotReady(t)
	cleanReports(t, projectDir)

	binPath := buildBinary(t)
	testPath := filepath.Join(projectDir, "tests", "not_tests")

	out, exitCode := runBinary(t, binPath, godotPath, testPath)

	// gdUnit4 behaviour when no test suites are found is implementation-dependent:
	// it may return 0 with 0 tests, or exit non-zero. Accept either outcome.
	switch exitCode {
	case 0:
		// Acceptable: Godot ran but found nothing to test.
		t.Logf("exit 0: total=%d status=%s", out.Summary.Total, out.Summary.Status)
	case 2:
		// Acceptable: gdUnit4 errored because no test suite was found.
		t.Logf("exit 2 (crash/error): status=%s", out.Summary.Status)
	default:
		t.Errorf("unexpected exit code %d for not_tests scenario", exitCode)
	}
}
