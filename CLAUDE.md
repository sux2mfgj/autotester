# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the project
go build -o perf-runner

# Run with configuration
./perf-runner -config examples/ib_send_bw-config.yaml
./perf-runner -config examples/iperf3-config.yaml

# JSON output with verbose logging
./perf-runner -json -verbose -config mytest.yaml

# Run with environment collection (enabled in config file)
./perf-runner -config mytest.yaml
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

- **EnvInfo** (`envinfo/`): Environment information collection
  - Gathers system information from test hosts
  - Collects NIC info, kernel version, software versions
  - Supports both local and remote collection via SSH

### Key Design Patterns

- **Registry Pattern**: Runners auto-register via `init()` functions
- **Strategy Pattern**: `Runner` interface allows different test tools
- **Template Method**: Standardized test execution flow in coordinator

### Data Flow

1. CLI loads YAML configuration and validates it
2. Coordinator establishes SSH connections to all hosts
3. For each test scenario:
   - **2-node topology**: Server starts → Client connects → Results collected
   - **3-node topology**: Server starts → Intermediate starts → Client connects → Results collected
4. Results formatted as JSON or human-readable text

## Configuration Structure

Tests are defined in YAML with these main sections:

- **Hosts**: SSH connection details and roles
- **Tests**: Client-server test scenarios
- **Binary Paths**: Custom paths for test tools (optional)
- **Runner-specific**: Tool parameters (duration, args, etc.)

## Environment Information Collection

The `-env` flag enables collection of comprehensive environment information from all test hosts:

**Collected Information:**
- **System**: Hostname, kernel version, OS info, architecture
- **CPU**: Model, cores, threads, frequency
- **Memory**: Total, available, used memory
- **Network Interfaces**: Name, IP addresses, MAC address, MTU, status, speed, driver
- **Software Versions**: ib_send_bw, iperf3, DPDK, socat, SSH versions

**Configuration:**
```yaml
name: "My Test"
runner: "iperf3"
collect_env: true  # Enable environment collection

hosts:
  # ... host definitions
tests:
  # ... test definitions
```

**Usage:**
```bash
# Run with environment collection (if enabled in config)
./perf-runner -config config.yaml

# Combine with JSON output for detailed environment data
./perf-runner -json -config config.yaml
```

**Output:** Environment information is included in test results under the `environment_info` field with separate sections for client, server, and intermediate hosts.

Example configuration with intermediate node:
```yaml
name: "Three-Node Test"
runner: "iperf3"

# Optional: specify custom paths for test binaries
binary_paths:
  iperf3: "/usr/bin/iperf3"
  socat: "/usr/bin/socat"  # For intermediate forwarding

hosts:
  server_host:
    ssh:
      host: "192.168.1.100"
      user: "testuser" 
      key_path: "~/.ssh/id_rsa"
    role: "server"
    
  forwarder_host:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "intermediate"
    
  client_host:
    ssh:
      host: "192.168.1.102"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"

tests:
  - name: "Test via Intermediate"
    client: "client_host"
    server: "server_host"
    intermediate: "forwarder_host"  # NEW: 3-node topology
    config:
      duration: 30s
      
  - name: "Direct Test"
    client: "client_host"
    server: "server_host"
    # No intermediate - 2-node topology
    config:
      duration: 30s
```

## Sudo Support

The tool supports configurable sudo execution for remote commands on a per-host basis. This is useful when performance tools require elevated privileges or when connecting with a user that needs sudo access.

**Configuration:**
Add `use_sudo: true` to the SSH configuration for any host that requires sudo:

```yaml
hosts:
  server_host:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
      use_sudo: true  # All commands on this host will be prefixed with 'sudo'
    role: "server"
    
  client_host:
    ssh:
      host: "192.168.1.101"
      user: "rootuser"  
      key_path: "~/.ssh/id_rsa"
      use_sudo: false  # Optional: explicitly disable sudo (default is false)
    role: "client"
```

**Requirements:**
- The target user must be configured for passwordless sudo access
- The user should have sudo privileges for the performance testing tools being used
- SSH key-based authentication is recommended for seamless operation

**Use Cases:**
- Performance tools requiring root privileges (e.g., system tuning, low-level hardware access)
- Non-root users who need elevated privileges for specific testing operations
- Mixed environments where some hosts require sudo and others don't

See `examples/example-sudo-support.yaml` for a complete example configuration.

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
- Binary output: `perf-runner` (after build)

## Dependencies

- Go 1.21+ required
- `golang.org/x/crypto/ssh` for SSH connections
- `gopkg.in/yaml.v3` for configuration parsing
- Target hosts must have test tools installed (`perftest` suite, `iperf3`)