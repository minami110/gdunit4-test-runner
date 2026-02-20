package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/minami110/gdunit4-test-runner/internal/config"
	"github.com/minami110/gdunit4-test-runner/internal/detector"
	"github.com/minami110/gdunit4-test-runner/internal/report"
	"github.com/minami110/gdunit4-test-runner/internal/runner"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		if errors.Is(err, config.ErrVersion) {
			fmt.Fprintln(os.Stderr, "gdunit4-test-runner", version)
			return 0
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	detected, err := detector.Detect(cfg.TestPaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	result, err := runner.Run(cfg.GodotPath, detected.ProjectDir, detected.ResPaths, cfg.Verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	defer os.Remove(result.LogFile)

	// Detect crashes in the Godot output log.
	crash, err := report.DetectCrash(result.LogFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	// If the process crashed (non-zero exit without a parseable report), emit crash-only JSON.
	xmlPath, xmlErr := report.FindReportXML(detected.ProjectDir)
	if xmlErr != nil {
		// No XML report found — emit crash/error output and exit.
		out := report.BuildOutput(nil, crash)
		if writeErr := report.WriteJSON(os.Stdout, out); writeErr != nil {
			fmt.Fprintln(os.Stderr, "error:", writeErr)
		}
		if crash != nil {
			return 2
		}
		// Godot ran but produced no report (unexpected).
		fmt.Fprintln(os.Stderr, "warning: Godot produced no test report")
		return 2
	}

	suites, err := report.ParseXML(xmlPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	out := report.BuildOutput(suites, crash)
	if err := report.WriteJSON(os.Stdout, out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	// Determine exit code based on results.
	switch out.Summary.Status {
	case "crashed":
		return 2
	case "failed":
		return 1
	default:
		return 0
	}
}
