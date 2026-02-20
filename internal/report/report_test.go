package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseXML_MixedResults(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_results.xml")
	suites, err := ParseXML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suites.Tests == 0 {
		t.Error("expected non-zero total tests")
	}
	if suites.Failures == 0 {
		t.Error("expected non-zero failures in sample_results.xml")
	}
}

func TestParseXML_AllPass(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_results_allpass.xml")
	suites, err := ParseXML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suites.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", suites.Failures)
	}
}

func TestParseXML_NotFound(t *testing.T) {
	_, err := ParseXML("/nonexistent/results.xml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestExtractFailures(t *testing.T) {
	suites := &JUnitTestSuites{
		Suites: []JUnitTestSuite{
			{
				TestCases: []JUnitTestCase{
					{
						Name:      "test_something",
						Classname: "MyTestClass",
						Failure: &JUnitFailure{
							Message: "FAILED: res://tests/MyTest.gd:42",
							Text:    "Expected 'foo' but was 'bar'",
						},
					},
					{
						Name:      "test_pass",
						Classname: "MyTestClass",
					},
				},
			},
		},
	}

	failures := ExtractFailures(suites)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	f := failures[0]
	if f.Class != "MyTestClass" {
		t.Errorf("Class = %q, want MyTestClass", f.Class)
	}
	if f.Method != "test_something" {
		t.Errorf("Method = %q, want test_something", f.Method)
	}
	if f.File != "res://tests/MyTest.gd" {
		t.Errorf("File = %q, want res://tests/MyTest.gd", f.File)
	}
	if f.Line != 42 {
		t.Errorf("Line = %d, want 42", f.Line)
	}
	if f.Expected != "foo" {
		t.Errorf("Expected = %q, want foo", f.Expected)
	}
	if f.Actual != "bar" {
		t.Errorf("Actual = %q, want bar", f.Actual)
	}
}

func TestExtractFailures_ErrorElement(t *testing.T) {
	suites := &JUnitTestSuites{
		Suites: []JUnitTestSuite{
			{
				TestCases: []JUnitTestCase{
					{
						Name:      "test_error",
						Classname: "ErrorClass",
						Error: &JUnitFailure{
							Message: "FAILED: res://tests/ErrorTest.gd:10",
						},
					},
				},
			},
		},
	}

	failures := ExtractFailures(suites)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure from error element, got %d", len(failures))
	}
	if failures[0].File != "res://tests/ErrorTest.gd" {
		t.Errorf("File = %q, want res://tests/ErrorTest.gd", failures[0].File)
	}
}

func TestDetectCrash_NoCrash(t *testing.T) {
	f, err := os.CreateTemp("", "no-crash-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("Godot Engine v4.2 - https://godotengine.org\nAll tests passed.\n")
	f.Close()

	result, err := DetectCrash(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil crash details, got %+v", result)
	}
}

func TestDetectCrash_WithCrash(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_crash.log")
	result, err := DetectCrash(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected crash details, got nil")
	}
	if !strings.Contains(result.CrashInfo, "handle_crash:") {
		t.Errorf("CrashInfo should contain 'handle_crash:', got: %q", result.CrashInfo)
	}
	if !strings.Contains(result.ScriptErrors, "SCRIPT ERROR:") {
		t.Errorf("ScriptErrors should contain 'SCRIPT ERROR:', got: %q", result.ScriptErrors)
	}
	if !strings.Contains(result.EngineErrors, "ERROR:") {
		t.Errorf("EngineErrors should contain 'ERROR:', got: %q", result.EngineErrors)
	}
}

func TestDetectCrash_NotFound(t *testing.T) {
	_, err := DetectCrash("/nonexistent/log.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestBuildOutput_AllPass(t *testing.T) {
	suites := &JUnitTestSuites{
		Tests:    5,
		Failures: 0,
		Errors:   0,
	}

	out := BuildOutput(suites, nil)
	if out.Summary.Total != 5 {
		t.Errorf("Total = %d, want 5", out.Summary.Total)
	}
	if out.Summary.Passed != 5 {
		t.Errorf("Passed = %d, want 5", out.Summary.Passed)
	}
	if out.Summary.Failed != 0 {
		t.Errorf("Failed = %d, want 0", out.Summary.Failed)
	}
	if out.Summary.Crashed {
		t.Error("Crashed should be false")
	}
	if out.Summary.Status != "passed" {
		t.Errorf("Status = %q, want passed", out.Summary.Status)
	}
}

func TestBuildOutput_WithFailures(t *testing.T) {
	suites := &JUnitTestSuites{
		Tests:    10,
		Failures: 2,
		Errors:   0,
	}

	out := BuildOutput(suites, nil)
	if out.Summary.Status != "failed" {
		t.Errorf("Status = %q, want failed", out.Summary.Status)
	}
	if out.Summary.Failed != 2 {
		t.Errorf("Failed = %d, want 2", out.Summary.Failed)
	}
	if out.Summary.Passed != 8 {
		t.Errorf("Passed = %d, want 8", out.Summary.Passed)
	}
}

func TestBuildOutput_Crashed(t *testing.T) {
	crash := &CrashDetails{CrashInfo: "handle_crash: signal 11"}
	out := BuildOutput(nil, crash)

	if !out.Summary.Crashed {
		t.Error("Crashed should be true")
	}
	if out.Summary.Status != "crashed" {
		t.Errorf("Status = %q, want crashed", out.Summary.Status)
	}
	if out.CrashDetails == nil {
		t.Error("CrashDetails should not be nil")
	}
}

func TestWriteJSON(t *testing.T) {
	out := &Output{
		Summary: Summary{
			Total:   3,
			Passed:  2,
			Failed:  1,
			Crashed: false,
			Status:  "failed",
		},
		Failures: []Failure{
			{Class: "Foo", Method: "test_bar", File: "res://foo.gd", Line: 10},
		},
	}

	var sb strings.Builder
	if err := WriteJSON(&sb, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed Output
	if err := json.Unmarshal([]byte(sb.String()), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.Summary.Total != 3 {
		t.Errorf("parsed Total = %d, want 3", parsed.Summary.Total)
	}
	if len(parsed.Failures) != 1 {
		t.Errorf("parsed Failures len = %d, want 1", len(parsed.Failures))
	}
}

func TestFindReportXML(t *testing.T) {
	root := t.TempDir()
	reportDir := filepath.Join(root, "reports", "report_20240101_120000")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	xmlPath := filepath.Join(reportDir, "results.xml")
	if err := os.WriteFile(xmlPath, []byte("<testsuites/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := FindReportXML(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != xmlPath {
		t.Errorf("found = %q, want %q", found, xmlPath)
	}
}

func TestFindReportXML_NotFound(t *testing.T) {
	root := t.TempDir()
	_, err := FindReportXML(root)
	if err == nil {
		t.Fatal("expected error when no report found, got nil")
	}
}
