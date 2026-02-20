# CLAUDE.md

This file provides guidance for AI assistants working on this repository.

## Project Overview

`gdunit4-test-runner` is a Go CLI tool that wraps the [gdUnit4](https://github.com/MikeSchulze/gdUnit4) test framework for Godot Engine. It is a single-purpose tool: discover the Godot project root, resolve the test path to a `res://` path, and execute Godot with gdUnit4's `GdUnitCmdTool.gd`.

**Not a framework. Not a library. Just a focused CLI binary.**

## Architecture

Four-package layout:

```
cmd/gdunit4-test-runner/
  main.go              # Entry point: parse config, run detector + runner, exit

internal/config/
  config.go            # Config struct, CLI flag parsing, env var reading, validation

internal/detector/
  detector.go          # Walk up from --path to find project.godot, verify addons/gdUnit4, convert to res:// path

internal/runner/
  runner.go            # Build Godot command arguments, exec process, stream stdout/stderr, return exit code
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
- Returns `(projectDir string, resPath string, error)`

**`internal/runner`**
- Accepts `Config` and the detector's results
- Constructs the Godot command: `godot --path <projectDir> -s -d res://addons/gdUnit4/bin/GdUnitCmdTool.gd -a <resPath>`
- Executes via `os/exec.Cmd`
- Pipes stdout and stderr to the caller's stdout/stderr in real time
- Returns the process exit code (does NOT wrap it)

**`cmd/gdunit4-test-runner/main.go`**
- Calls config, detector, runner in sequence
- On any error from config or detector, prints error to stderr and exits with code 1
- On runner completion, exits with the exact code returned by runner

## Key Design Decisions

### No external dependencies

Only the Go standard library. No Cobra, no Viper, no third-party packages. This keeps the binary small and the dependency surface zero.

### Standard `flag` package

Cobra is overkill for a single-subcommand tool. Use `flag.StringVar`, `flag.BoolVar`, etc.

### Exit code passthrough

gdUnit4 uses exit code 0 (pass), 100 (failure), 101 (error). The runner must pass these through unchanged. Exit code 1 is reserved for tool-level errors (Godot not found, project not found, etc.).

### Real-time output streaming

Do not buffer Godot's output. Connect `cmd.Stdout = os.Stdout` and `cmd.Stderr = os.Stderr` directly so users see test results as they happen.

### Godot binary resolution order

1. `--godot-path` CLI flag
2. `GODOT_PATH` environment variable
3. `godot` found via `exec.LookPath("godot")`

If none resolve to an executable file, return an error from config validation.

### Project detection

Starting from the absolute path of `--path`, walk up parent directories until `project.godot` is found or the filesystem root is reached. If not found, return an error. Then check that `<projectDir>/addons/gdUnit4/` exists. Convert `--path` to `res://` by stripping `projectDir` prefix and prepending `res://`.

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

### Testing conventions

- Table-driven tests (`[]struct{ name, input, want }`)
- Use `t.TempDir()` for filesystem fixtures in detector tests
- No mocking frameworks — use interfaces only where genuinely needed

## Important Patterns

### Godot command construction

```go
args := []string{
    "--path", projectDir,
    "-s", "-d",
    "res://addons/gdUnit4/bin/GdUnitCmdTool.gd",
    "-a", resPath,
}
if cfg.ContinueOnFailure {
    args = append(args, "--continue-on-failure")
}
cmd := exec.Command(cfg.GodotPath, args...)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

### res:// path conversion

```go
// projectDir: /home/user/myproject
// testPath:   /home/user/myproject/tests/unit
// result:     res://tests/unit
rel, err := filepath.Rel(projectDir, testPath)
resPath := "res://" + filepath.ToSlash(rel)
```
