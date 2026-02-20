package report

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ---- XML structures (gdUnit4 JUnit XML format) ----

// JUnitTestSuites represents the root <testsuites> element.
type JUnitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Suites   []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a <testsuite> element.
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Package   string          `xml:"package,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a <testcase> element.
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Failure   *JUnitFailure `xml:"failure"`
	Error     *JUnitFailure `xml:"error"`
}

// JUnitFailure represents a <failure> or <error> element.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// ---- JSON output structures ----

// Output is the top-level JSON output.
type Output struct {
	Summary      Summary       `json:"summary"`
	CrashDetails *CrashDetails `json:"crash_details,omitempty"`
	Failures     []Failure     `json:"failures"`
}

// Summary holds test result counts and overall status.
type Summary struct {
	Total   int    `json:"total"`
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Crashed bool   `json:"crashed"`
	Status  string `json:"status"` // "passed", "failed", or "crashed"
}

// CrashDetails holds crash/error information extracted from the Godot log.
type CrashDetails struct {
	CrashInfo    string `json:"crash_info,omitempty"`
	ScriptErrors string `json:"script_errors,omitempty"`
}

// Failure represents a single test failure.
type Failure struct {
	Class    string `json:"class"`
	Method   string `json:"method"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
}

// ---- Regex patterns ----

// failedLocRe matches "FAILED: res://path/to/file.gd:42" in failure messages.
var failedLocRe = regexp.MustCompile(`FAILED:\s*(res://[^:]+):(\d+)`)

// expectedActualRe matches "Expected '<x>' but was '<y>'" patterns in CDATA.
var expectedActualRe = regexp.MustCompile(`Expected\s+'([^']*)'\s+but was\s+'([^']*)'`)

// ---- Public API ----

// FindReportXML finds the most recently modified results.xml under projectDir/reports/report_*/.
func FindReportXML(projectDir string) (string, error) {
	pattern := filepath.Join(projectDir, "reports", "report_*", "results.xml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for report files: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no report file found matching: %s", pattern)
	}

	// Return the most recently modified file.
	newest := matches[0]
	newestInfo, err := os.Stat(newest)
	if err != nil {
		return "", err
	}
	for _, m := range matches[1:] {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestInfo.ModTime()) {
			newest = m
			newestInfo = info
		}
	}
	return newest, nil
}

// ParseXML parses a JUnit XML file produced by gdUnit4.
func ParseXML(path string) (*JUnitTestSuites, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open XML file: %w", err)
	}
	defer f.Close()

	var suites JUnitTestSuites
	if err := xml.NewDecoder(f).Decode(&suites); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	return &suites, nil
}

// ExtractFailures extracts Failure entries from parsed test suites.
func ExtractFailures(suites *JUnitTestSuites) []Failure {
	var failures []Failure
	for _, suite := range suites.Suites {
		for _, tc := range suite.TestCases {
			f := tc.Failure
			if f == nil {
				f = tc.Error
			}
			if f == nil {
				continue
			}
			failure := Failure{
				Class:   tc.Classname,
				Method:  tc.Name,
				Message: f.Message,
			}
			// Extract file and line from the message (e.g. "FAILED: res://path.gd:42").
			if m := failedLocRe.FindStringSubmatch(f.Message); m != nil {
				failure.File = m[1]
				if line, err := strconv.Atoi(m[2]); err == nil {
					failure.Line = line
				}
			}
			// Extract expected/actual from CDATA body (best-effort).
			body := strings.TrimSpace(f.Text)
			if m := expectedActualRe.FindStringSubmatch(body); m != nil {
				failure.Expected = m[1]
				failure.Actual = m[2]
			}
			failures = append(failures, failure)
		}
	}
	return failures
}

// DetectCrash scans the Godot log file for crash/error patterns.
// Returns nil if no crash indicators are found.
func DetectCrash(logPath string) (*CrashDetails, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var crashLines []string
	var scriptErrorLines []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.Contains(line, "handle_crash:"):
			crashLines = append(crashLines, line)
		case strings.HasPrefix(line, "SCRIPT ERROR:"):
			scriptErrorLines = append(scriptErrorLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	if len(crashLines) == 0 && len(scriptErrorLines) == 0 {
		return nil, nil
	}

	return &CrashDetails{
		CrashInfo:    strings.Join(crashLines, "\n"),
		ScriptErrors: strings.Join(scriptErrorLines, "\n"),
	}, nil
}

// BuildOutput constructs the Output struct from parsed suites and optional crash details.
func BuildOutput(suites *JUnitTestSuites, crash *CrashDetails) *Output {
	failures := []Failure{}
	if suites != nil {
		extracted := ExtractFailures(suites)
		if extracted != nil {
			failures = extracted
		}
	}

	crashed := crash != nil
	total := 0
	failed := 0
	if suites != nil {
		total = suites.Tests
		failed = suites.Failures + suites.Errors
	}
	passed := total - failed
	if passed < 0 {
		passed = 0
	}

	status := "passed"
	if crashed {
		status = "crashed"
	} else if failed > 0 {
		status = "failed"
	}

	return &Output{
		Summary: Summary{
			Total:   total,
			Passed:  passed,
			Failed:  failed,
			Crashed: crashed,
			Status:  status,
		},
		CrashDetails: crash,
		Failures:     failures,
	}
}

// WriteJSON encodes the Output as indented JSON to w.
func WriteJSON(w io.Writer, out *Output) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	return nil
}
