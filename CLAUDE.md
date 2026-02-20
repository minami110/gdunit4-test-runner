# CLAUDE.md

This file provides guidance for AI assistants working on this repository.

## Project Overview

`gdunit4-test-runner` is a Go CLI tool that wraps the [gdUnit4](https://github.com/MikeSchulze/gdUnit4) test framework for Godot Engine. It discovers the Godot project root, executes tests via `GdUnitCmdTool.gd`, parses the JUnit XML report, and outputs structured JSON results to stdout.

**Not a framework. Not a library. Just a focused CLI binary.**

## Architecture

Five-package layout:

```
cmd/gdunit4-test-runner/
  main.go              # Entry point: parse config, run detector + runner + report, exit

internal/config/
  config.go            # Config struct, CLI flag parsing, env var reading, validation

internal/detector/
  detector.go          # Walk up from --path to find project.godot, verify addons/gdUnit4, convert to res:// path

internal/runner/
  runner.go            # Build Godot command arguments, exec process, capture output to temp file, return exit code

internal/report/
  report.go            # Find and parse JUnit XML, detect crashes in log, build and write JSON output
```

### Package responsibilities

**`internal/config`**
- Defines `Config` struct holding all runtime settings
- Parses CLI flags using the standard `flag` package
- Reads `GODOT_PATH` environment variable
- Validates required fields and resolves Godot binary path
- Returns error for missing or invalid configuration

**`internal/detector`**
- Accepts a filesystem path (absolute or relative)
- Walks up the directory tree looking for `project.godot`
- Verifies `addons/gdUnit4/` exists at the project root
- Converts the original test path to a `res://`-relative path
- Returns `*Result{ ProjectDir, ResPath }` or error

**`internal/runner`**
- Accepts godotPath, projectDir, resPath, verbose
- Constructs the Godot command: `godot --headless -s -d res://addons/gdUnit4/bin/GdUnitCmdTool.gd -a <resPath> --ignoreHeadlessMode -c`
- Sets `cmd.Dir = projectDir` (runs from project root, not `--path`)
- Captures stdout+stderr to a temp log file
- If verbose, tees output to stderr via `io.MultiWriter`
- Returns `*RunResult{ ExitCode, LogFile }` — caller owns the log file

**`internal/report`**
- `FindReportXML(projectDir)` — globs `reports/report_*/results.xml`, returns newest
- `ParseXML(path)` — decodes JUnit XML via `encoding/xml`
- `ExtractFailures(suites)` — extracts file/line from failure message, expected/actual from CDATA
- `DetectCrash(logPath)` — line-by-line scan for `handle_crash:`, `SCRIPT ERROR:`, `ERROR:` prefixes
- `BuildOutput(suites, crash)` — constructs `Output` struct with summary + failures
- `WriteJSON(w, out)` — `json.Encoder` with `SetIndent("", "  ")`

**`cmd/gdunit4-test-runner/main.go`**
- Calls config → detector → runner → report in sequence
- JSON goes to stdout only; all other messages go to stderr
- `defer os.Remove(result.LogFile)` for temp file cleanup
- Exit codes: 0 (passed), 1 (failed), 2 (crashed / tool error)

## Key Design Decisions

### No external dependencies

Only the Go standard library. No Cobra, no Viper, no third-party packages.

### Standard `flag` package

Use `flag.NewFlagSet` + `ContinueOnError` for testability. Support both `-v` and `--verbose` by binding two flags to the same bool variable.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | All tests passed |
| `1` | Test failure(s) detected |
| `2` | Crash, tool error, or Godot not found |

### Output separation

- **stdout**: JSON result only
- **stderr**: Verbose Godot output (when `-v`), error messages

### Temp log file ownership

`runner.Run` creates a temp file and returns its path. `main.go` owns cleanup via `defer os.Remove`. This allows the report package to read the file after `runner.Run` returns.

### Godot execution

```go
cmd := exec.Command(godotPath, "--headless", "-s", "-d",
    "res://addons/gdUnit4/bin/GdUnitCmdTool.gd",
    "-a", resPath, "--ignoreHeadlessMode", "-c")
cmd.Dir = projectDir  // run from project root
```

### Godot binary resolution order

1. `--godot-path` CLI flag
2. `GODOT_PATH` environment variable
3. `godot` found via `exec.LookPath("godot")`

If none resolve to an executable file, return an error from config validation.

### Project detection

Starting from the absolute path of `--path`, walk up parent directories until `project.godot` is found or the filesystem root is reached. Then check that `<projectDir>/addons/gdUnit4/` exists. Convert `--path` to `res://` using `filepath.Rel` + `filepath.ToSlash`.

## Development

### Requirements

- Go 1.24+

### Common commands

```sh
go build ./cmd/gdunit4-test-runner   # Build binary
go test ./...                         # Run all tests
go vet ./...                          # Lint
gofmt -w .                            # Format
```

Or via Makefile: `make build`, `make test`, `make lint`, `make fmt`

### Testing conventions

- Table-driven tests (`[]struct{ name, input, want }`)
- Use `t.TempDir()` for filesystem fixtures in detector tests
- No mocking frameworks — use interfaces only where genuinely needed
- Testdata fixtures in `testdata/`: XML reports and crash logs for report package tests

### Sandbox testing

Place a Godot project in `sandbox/` (gitignored) for manual end-to-end testing:

```sh
./gdunit4-test-runner --path sandbox/tests/ --godot-path /path/to/godot -v
./gdunit4-test-runner --path sandbox/tests/ | jq .
```

## Important Patterns

### res:// path conversion

```go
// projectDir: /home/user/myproject
// testPath:   /home/user/myproject/tests/unit
// result:     res://tests/unit
rel, err := filepath.Rel(projectDir, testPath)
resPath := "res://" + filepath.ToSlash(rel)
```

### Crash detection

```go
switch {
case strings.Contains(line, "handle_crash:"):
    // crash signal
case strings.HasPrefix(line, "SCRIPT ERROR:"):
    // GDScript error
case strings.HasPrefix(line, "ERROR:"):
    // engine error
}
```

### Failure message parsing

```go
// From XML: <failure message="FAILED: res://tests/Foo.gd:42">
var failedLocRe = regexp.MustCompile(`FAILED:\s*(res://[^:]+):(\d+)`)

// From CDATA: Expected 'foo' but was 'bar'
var expectedActualRe = regexp.MustCompile(`Expected\s+'([^']*)'\s+but was\s+'([^']*)'`)
```
