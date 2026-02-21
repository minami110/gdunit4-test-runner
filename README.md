# gdunit4-test-runner

A CLI tool that wraps [gdUnit4](https://github.com/MikeSchulze/gdUnit4) test framework for Godot Engine. It discovers the Godot project root, executes tests via `GdUnitCmdTool.gd`, and outputs structured JSON results.

## Features

- **Single binary** — no runtime dependencies, just download and run
- **Cross-platform** — Linux and Windows support
- **Auto-detection** — automatically finds `project.godot` by walking up from the given path
- **JSON output** — machine-readable test results on stdout for easy CI integration
- **Verbose mode** — optionally stream raw Godot output to stderr while JSON goes to stdout

## Installation

### From Releases

Download the latest binary from the [Releases](https://github.com/minami110/gdunit4-test-runner/releases) page.

### From Source

```sh
go install github.com/minami110/gdunit4-test-runner/cmd/gdunit4-test-runner@latest
```

## Usage

### Basic

```sh
# Run all tests under tests/ (current directory is used if no path given)
gdunit4-test-runner tests/

# Run tests with a specific Godot binary
gdunit4-test-runner --godot-path /usr/local/bin/godot4 tests/

# Run multiple paths at once
gdunit4-test-runner tests/unit tests/integration

# Run a single test file
gdunit4-test-runner tests/MyTest.gd

# Run tests and stream Godot output to stderr while JSON goes to stdout
gdunit4-test-runner --verbose tests/

# Use current directory (omit path entirely)
gdunit4-test-runner --godot-path /usr/local/bin/godot4

# Parse JSON output with jq
gdunit4-test-runner tests/ | jq .summary
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `[paths...]` | `.` (current dir) | One or more paths to test directories or files (relative or absolute) |
| `--godot-path` | *(auto)* | Path to Godot binary. Overrides `GODOT_PATH` env and PATH lookup |
| `--verbose` | `false` | Stream raw Godot output to stderr |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GODOT_PATH` | Path to Godot binary. Used when `--godot-path` is not specified |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All tests passed |
| `1` | Test failure(s) detected |
| `2` | Crash, tool error, or Godot not found |

## JSON Output Format

```json
{
  "summary": {
    "total": 10,
    "passed": 8,
    "failed": 2,
    "crashed": false,
    "status": "failed"
  },
  "crash_details": null,
  "failures": [
    {
      "class": "TestClass",
      "method": "test_method",
      "file": "res://tests/TestClass.gd",
      "line": 42,
      "expected": "foo",
      "actual": "bar",
      "message": "FAILED: res://tests/TestClass.gd:42"
    }
  ]
}
```

**`summary.status`** is one of:
- `"passed"` — all tests passed
- `"failed"` — one or more test failures
- `"crashed"` — Godot crashed or a script error occurred

## How It Works

1. **Project detection**: Starting from the first given path, walks up the directory tree to find `project.godot`. Also verifies that `addons/gdUnit4/` is present.
2. **Path conversion**: Converts each filesystem path to a `res://`-relative path.
3. **Execution**: Runs Godot from the project directory:
   ```
   godot --headless -s res://addons/gdUnit4/bin/GdUnitCmdTool.gd -a <res://path1> -a <res://path2> --ignoreHeadlessMode -c
   ```
4. **Output capture**: Captures Godot stdout+stderr to a temp log file; if `--verbose` is set, also tees to stderr.
5. **Crash detection**: Scans the log for `handle_crash:`, `SCRIPT ERROR:`, and `ERROR:` patterns.
6. **Report parsing**: Reads `reports/report_*/results.xml` (JUnit XML) produced by gdUnit4.
7. **JSON output**: Writes structured results to stdout.

### Godot Binary Resolution Order

1. `--godot-path` flag
2. `GODOT_PATH` environment variable
3. `godot` on `PATH`

## Build

### Prerequisites

- Go 1.24+

### Commands

```sh
make build          # Build for current platform
make build-linux    # Build for Linux (amd64)
make build-windows  # Build for Windows (amd64)
make test           # Run tests
make lint           # Run go vet
make fmt            # Format code
```

## License

MIT — see [LICENSE](LICENSE)
