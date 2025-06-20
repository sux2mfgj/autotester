# Perftest Tester - InfiniBand Performance Testing Tool

A Go-based tool for executing InfiniBand performance tests using perftest suite tools (like ib_send_bw) across multiple hosts via SSH connections. This tool coordinates client-server tests between remote machines and provides comprehensive result reporting.

## Features

- 🚀 **Multi-host Testing**: Execute tests across multiple remote hosts via SSH
- 🔌 **Extensible Architecture**: Plugin-based runner system for different test tools
- 📊 **Rich Metrics**: Extract and parse performance metrics from test outputs
- 🔧 **Configurable**: YAML-based configuration for hosts and test scenarios
- 📝 **Multiple Output Formats**: JSON and human-readable text output
- 🔒 **Security**: SSH key-based authentication
- ⚡ **Concurrent Execution**: Parallel host connections and coordinated test execution
- 🎯 **Test Scenarios**: Support for multiple test scenarios with repetition

## Quick Start

### Prerequisites

- Go 1.21 or later
- SSH access to target hosts
- `perftest` suite installed on target hosts (ib_send_bw, ib_read_bw, etc.)
- InfiniBand hardware and drivers configured

### Installation

```bash
git clone <repository-url>
cd tester
go mod tidy
go build -o tester
```

### Basic Usage

1. **Create a configuration file** (see `examples/ib_send_bw-config.yaml`):

```yaml
name: "InfiniBand Performance Test"
runner: "ib_send_bw"
timeout: 5m

hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      port: 18515

  ib_client:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"

tests:
  - name: "Basic IB Send BW Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        iterations: 1000
        connection: "RC"
```

2. **Run tests**:

```bash
# Use example configuration
./tester -config examples/ib_send_bw-config.yaml

# Use custom configuration
./tester -config mytest.yaml

# JSON output
./tester -json

# Verbose logging
./tester -verbose
```

## Configuration

### Host Configuration

Each host requires SSH connection details and role assignment:

```yaml
hosts:
  hostname:
    ssh:
      host: "IP_ADDRESS"
      port: 22                    # Optional, defaults to 22
      user: "username"
      key_path: "~/.ssh/id_rsa"  # SSH private key path
      # password: "password"      # Alternative to key_path
      connect_timeout: 30s
      command_timeout: 300s
    role: "client|server"         # Optional role hint
    runner:                       # Host-specific runner config
      port: 5201
      args:
        format: "m"
```

### Test Scenarios

Define multiple test scenarios with different parameters:

```yaml
tests:
  - name: "TCP Bandwidth Test"
    description: "Basic TCP performance test"
    client: "client_host"
    server: "server_host"
    config:
      duration: 30s
      args:
        parallel: 4
        window: "1M"
    repeat: 3                     # Run 3 times
    delay: 5s                     # 5s delay between runs
```

### Supported Test Tools

| Tool | Description | Use Case |
|------|-------------|----------|
| `ib_send_bw` | InfiniBand send bandwidth test | High-performance InfiniBand send testing |

### ib_send_bw Configuration

#### Network Configuration

| Field | Type | Description |
|-------|------|-------------|
| `target_host` | string | Specific IP address for client to connect to (overrides SSH host) |
| `port` | int | Port number for the test |

**Separate SSH and InfiniBand Networks:**

In many HPC environments, SSH management traffic and InfiniBand traffic use different networks. You can configure different IPs:

```yaml
hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"    # Management network for SSH
    runner:
      # Server doesn't need target_host (listens on all interfaces)
      
  ib_client:
    ssh:
      host: "192.168.1.101"    # Management network for SSH  
    runner:
      target_host: "10.0.0.100"  # InfiniBand network IP for testing
```

You can also override `target_host` per test:

```yaml
tests:
  - name: "Test with specific IB IP"
    client: "ib_client"
    server: "ib_server"
    config:
      target_host: "10.0.0.200"  # Override for this test
```

#### ib_send_bw Arguments

Supported ib_send_bw arguments through `config.args`:

| Argument | Type | Description |
|----------|------|-------------|
| `size` | int/string | Message size in bytes (e.g., 65536) |
| `iterations` | int | Number of iterations to run |
| `tx_depth` | int | Send queue depth |
| `rx_depth` | int | Receive queue depth |
| `mtu` | int | MTU size (e.g., 4096) |
| `qp` | int | Number of Queue Pairs |
| `connection` | string | Connection type (RC/UC/UD) |
| `inline` | int | Inline message size |
| `gid_index` | int | GID index to use |
| `sl` | int | Service level |
| `cpu_freq` | float | CPU frequency for cycle calculations |
| `use_event` | bool | Use event completion |
| `bidirectional` | bool | Bidirectional test |
| `report_cycles` | bool | Report CPU cycles |
| `report_histogram` | bool | Report latency histogram |
| `odp` | bool | Use On Demand Paging |
| `report_gbits` | bool | Report in Gb/sec instead of MB/sec |

## Command Line Options

```bash
Usage: ./tester [options]

Options:
  -config string
        Path to configuration file (default "config.yaml")
  -timeout duration
        Global timeout for all tests (default 10m0s)
  -verbose
        Enable verbose logging
  -json
        Output results in JSON format
  -version
        Show version information
```

## Output Examples

### Text Output
```
=== Test Results ===
Total Duration: 45.2s
Total Tests: 3
Passed: 2
Failed: 1

1. IB Send BW Test - Basic
   Status: ✓ PASS
   Duration: 12.3s
   Client: ✓ PASS
   Client Metrics:
     bandwidth_bps: 1.048576e+09
     bandwidth_readable: 1000.00 MB/sec
     message_rate_pps: 15625000
   Server: ✓ PASS

2. IB Send BW Test - Failed
   Status: ✗ FAIL
   Duration: 2.1s
   Client: ✗ FAIL
     Client Error: SSH command execution failed: Process exited with status 1
     Client Exit Code: 1
     Client Output:
       ib_send_bw: command not found
   Server: ✗ FAIL
     Server Error: SSH command execution failed: Process exited with status 127
     Server Exit Code: 127
     Server Output:
       bash: ib_send_bw: command not found
```

### JSON Output
```json
{
  "total_duration": "45.234567s",
  "total_tests": 3,
  "passed": 2,
  "failed": 1,
  "results": [
    {
      "scenario_name": "IB Send BW Test - Basic",
      "success": true,
      "duration": "12.345s",
      "client_result": {
        "success": true,
        "exit_code": 0,
        "duration": "12.1s",
        "metrics": {
          "bandwidth_bps": 1048576000,
          "bandwidth_readable": "1000.00 MB/sec",
          "message_rate_pps": 15625000
        }
      },
      "server_result": {
        "success": true,
        "exit_code": 0,
        "duration": "12.3s"
      }
    },
    {
      "scenario_name": "IB Send BW Test - Failed",
      "success": false,
      "duration": "2.1s",
      "client_result": {
        "success": false,
        "exit_code": 127,
        "duration": "2.0s",
        "error": "SSH command execution failed: Process exited with status 127",
        "output": "ib_send_bw: command not found"
      },
      "server_result": {
        "success": false,
        "exit_code": 127,
        "duration": "2.1s",
        "error": "SSH command execution failed: Process exited with status 127",
        "output": "bash: ib_send_bw: command not found"
      }
    }
  ]
}
```

## Architecture

The tool follows a modular architecture with clear separation of concerns:

- **CLI**: Command line interface and application orchestration
- **Config**: YAML configuration loading and validation
- **Coordinator**: Test execution orchestration across hosts
- **Runner**: Abstraction for different perftest tools (ib_send_bw, etc.)
- **SSH**: Remote connection and command execution
- **Output**: Result formatting and display

For detailed architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Extending the Tool

### Adding New Test Tools

1. Implement the `Runner` interface:

```go
type Runner interface {
    Run(ctx context.Context, config Config) (*Result, error)
    Validate(config Config) error
    Name() string
    SupportsRole(role string) bool
}
```

2. Add command building logic to `CommandBuilder`
3. Register the runner in `cli/app.go`

### Adding New Output Formats

1. Extend the `Formatter` in `output/formatter.go`
2. Add CLI flag for the new format
3. Implement format-specific logic

## Examples

The `examples/` directory contains:

- `ib_send_bw-config.yaml`: Complete InfiniBand bandwidth testing examples
- Various test scenario examples (different message sizes, connection types, performance analysis)

### InfiniBand Testing Example

```bash
# Run InfiniBand bandwidth tests
./tester -config examples/ib_send_bw-config.yaml

# Run with verbose output to see command execution
./tester -config examples/ib_send_bw-config.yaml -verbose
```

## Security Considerations

- Uses SSH key-based authentication (recommended)
- Password authentication supported but not recommended
- Host key verification should be implemented for production use
- No secrets stored in configuration files

## Troubleshooting

### Common Issues

1. **Connection refused**: Ensure SSH service is running on target hosts
2. **Permission denied**: Check SSH key permissions and user access
3. **Command not found**: Ensure perftest suite (ib_send_bw) is installed and in PATH on target hosts
4. **Timeout errors**: Increase timeout values in configuration
5. **InfiniBand errors**: Verify IB hardware is properly configured and active
6. **RDMA device not found**: Check that InfiniBand drivers and devices are available

### Debug Mode

Run with `-verbose` flag for detailed logging:

```bash
./tester -verbose -config debug.yaml
```

## Development

### Prerequisites
- Go 1.21+
- SSH access to test hosts

### Building
```bash
go build -o tester
```

### Testing
```bash
go test ./...
```

### Code Structure
```
tester/
├── main.go                 # Entry point
├── cli/                    # Command line interface
├── config/                 # Configuration management
├── coordinator/            # Test orchestration
├── runner/                 # Test tool abstraction
├── ssh/                    # SSH client
├── output/                 # Result formatting
└── examples/               # Example configurations
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license here]

## Support

For issues and questions:
- Check existing issues in the repository
- Create a new issue with detailed information
- Include configuration and log output for troubleshooting