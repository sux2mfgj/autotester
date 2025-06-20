# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the project
go build -o tester

# Run with configuration
./tester -config examples/ib_send_bw-config.yaml
./tester -config examples/iperf3-config.yaml

# JSON output with verbose logging
./tester -json -verbose -config mytest.yaml
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./config
go test ./runner
go test ./coordinator

# Run tests with verbose output
go test -v ./...

# Run specific test function
go test -run TestIbSendBwBuildCommand ./runner
```

### Development
```bash
# Install dependencies
go mod tidy

# Format code
go fmt ./...

# Check for issues
go vet ./...
```

## Architecture Overview

This is a Go-based InfiniBand/network performance testing tool that orchestrates tests across multiple remote hosts via SSH. The architecture follows a modular plugin-based design:

### Core Components

- **CLI** (`cli/`): Command-line interface and application orchestration
  - `app.go`: Main application logic and flow control
  - `flags.go`: Command line flag definitions

- **Config** (`config/`): YAML configuration management
  - Handles host definitions, SSH settings, and test scenarios
  - Validates configuration before execution

- **Coordinator** (`coordinator/`): Test execution orchestration
  - Manages SSH connections to multiple hosts
  - Coordinates client-server test execution
  - Handles parallel connections and synchronization

- **Runner** (`runner/`): Pluggable test tool abstraction
  - Plugin-based system with auto-registration
  - Currently supports `ib_send_bw` and `iperf3`
  - Each runner implements: `Validate()`, `BuildCommand()`, `ParseMetrics()`

- **SSH** (`ssh/`): Remote connection management
  - Key-based authentication
  - Remote command execution
  - Connection pooling and cleanup

- **Output** (`output/`): Result formatting (JSON/text)

### Key Design Patterns

- **Registry Pattern**: Runners auto-register via `init()` functions
- **Strategy Pattern**: `Runner` interface allows different test tools
- **Template Method**: Standardized test execution flow in coordinator

### Data Flow

1. CLI loads YAML configuration and validates it
2. Coordinator establishes SSH connections to all hosts
3. For each test scenario:
   - Server runner starts in background
   - Client runner connects and executes test
   - Results are collected and parsed for metrics
4. Results formatted as JSON or human-readable text

## Configuration Structure

Tests are defined in YAML with three main sections:

- **Hosts**: SSH connection details and roles
- **Tests**: Client-server test scenarios
- **Runner-specific**: Tool parameters (duration, args, etc.)

Example host configuration:
```yaml
hosts:
  server_host:
    ssh:
      host: "192.168.1.100"
      user: "testuser" 
      key_path: "~/.ssh/id_rsa"
    role: "server"
```

## Adding New Test Tools

To add support for a new performance test tool:

1. Create new file in `runner/` package (e.g., `my_tool.go`)
2. Implement the `Runner` interface:
   - `Name()`: Return tool name
   - `SupportsRole()`: Return if tool supports client/server roles
   - `BuildCommand()`: Build command line from config
   - `ParseMetrics()`: Extract performance metrics from output
   - `Validate()`: Validate configuration
3. Add auto-registration in `init()` function:
   ```go
   func init() {
       Register("my_tool", func() Runner { return &MyToolRunner{} })
   }
   ```

The runner will be automatically discovered and available for use.

## Common File Locations

- Example configurations: `examples/`
- Documentation: `docs/` (USER_GUIDE.md, DEVELOPER_GUIDE.md, etc.)
- Test files: `*_test.go` files throughout packages
- Binary output: `tester` (after build)

## Dependencies

- Go 1.21+ required
- `golang.org/x/crypto/ssh` for SSH connections
- `gopkg.in/yaml.v3` for configuration parsing
- Target hosts must have test tools installed (`perftest` suite, `iperf3`)