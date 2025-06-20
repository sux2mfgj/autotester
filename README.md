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
      port: 18515
      args:
        size: 65536
        iterations: 1000
        ib_dev: "mlx5_0"
        gid_index: 3
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
| `iperf3` | TCP/UDP network bandwidth test | General network performance testing |

For detailed runner parameter documentation, see [docs/RUNNER_PARAMETERS.md](docs/RUNNER_PARAMETERS.md).

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

## Output Formats

The tool supports both human-readable text output and structured JSON output:

- **Text Output**: Displays test results in a readable format with status, duration, metrics, and detailed error information
- **JSON Output**: Provides structured output suitable for parsing and integration with other tools

Use the `-json` flag to enable JSON output format. Both formats include:
- Test execution status and timing
- Command lines executed on each host  
- Complete stdout/stderr output from tests
- Parsed performance metrics
- Detailed error information for failed tests

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
    Validate(config Config) error
    Name() string
    SupportsRole(role string) bool
    BuildCommand(config Config) string
    ParseMetrics(result *Result) error
}
```

2. Add auto-registration in your runner's `init()` function
3. The runner will be automatically discovered by the CLI
4. Document parameters in [docs/RUNNER_PARAMETERS.md](docs/RUNNER_PARAMETERS.md)

For detailed guidance, see [docs/ADDING_NEW_COMMANDS.md](docs/ADDING_NEW_COMMANDS.md).

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
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific command builder tests
go test -v ./runner
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