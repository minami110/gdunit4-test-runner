# gdunit4-test-runner

A CLI tool that wraps [gdUnit4](https://github.com/MikeSchulze/gdUnit4) test framework for Godot Engine, enabling easy test execution from the command line and CI/CD pipelines.

## Features

- **Single binary** — no runtime dependencies, just download and run
- **Cross-platform** — Linux and Windows support
- **Auto-detection** — automatically finds `project.godot` by walking up from the given path
- **Exit code passthrough** — preserves gdUnit4 exit codes (0/100/101) for CI integration
- **Real-time output** — streams stdout/stderr from Godot process as it runs

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
# Run all tests under res://tests/
gdunit4-test-runner --path tests/

# Run tests with a specific Godot binary
gdunit4-test-runner --path tests/ --godot-path /usr/local/bin/godot4

# Run tests and continue on failure
gdunit4-test-runner --path tests/ --continue-on-failure
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | *(required)* | Path to test directory or file (relative or absolute) |
| `--godot-path` | *(auto)* | Path to Godot binary. Overrides `GODOT_PATH` env and PATH lookup |
| `--continue-on-failure` | `false` | Continue running tests after a failure |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GODOT_PATH` | Path to Godot binary. Used when `--godot-path` is not specified |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All tests passed |
| `1` | Tool error (missing Godot binary, project not found, etc.) |
| `100` | Test failure(s) detected |
| `101` | Test error(s) detected |

## How It Works

`gdunit4-test-runner` is a thin wrapper around gdUnit4's `GdUnitCmdTool.gd` script.

1. **Project detection**: Starting from `--path`, walks up the directory tree to find `project.godot`. Also verifies that `addons/gdUnit4/` is present.
2. **Path conversion**: Converts the filesystem path of the test target to a `res://`-relative path (required by Godot).
3. **Command construction**: Builds and executes the following command:
   ```
   godot --path <project-dir> -s -d res://addons/gdUnit4/bin/GdUnitCmdTool.gd -a <res://path/to/tests>
   ```
4. **Output streaming**: Pipes stdout/stderr in real time so test output appears immediately.

### Godot Binary Resolution Order

1. `--godot-path` flag
2. `GODOT_PATH` environment variable
3. `godot` on `PATH`

## Build

### Prerequisites

- Go 1.24+

### Build

```sh
go build ./cmd/gdunit4-test-runner
```

### Cross-compile

```sh
# Linux
GOOS=linux GOARCH=amd64 go build -o dist/gdunit4-test-runner ./cmd/gdunit4-test-runner

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/gdunit4-test-runner.exe ./cmd/gdunit4-test-runner
```

### Makefile

```sh
make build          # Build for current platform
make build-linux    # Build for Linux (amd64)
make build-windows  # Build for Windows (amd64)
make test           # Run tests
make lint           # Run go vet
```

## License

MIT — see [LICENSE](LICENSE)
