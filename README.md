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

## Documentation

### For Users
- **[User Guide](docs/USER_GUIDE.md)** - Complete guide for running tests, configuration, and troubleshooting
- **[Tool Parameters](docs/RUNNER_PARAMETERS.md)** - Detailed parameter reference for all supported tools

### For Developers  
- **[Developer Guide](docs/DEVELOPER_GUIDE.md)** - Architecture overview, development setup, and contribution guidelines
- **[Extending Runners](docs/EXTENDING_RUNNERS.md)** - Detailed guide for adding new test tools

## Quick Start

### Prerequisites
- Go 1.21 or later
- SSH access to target hosts  
- Test tools installed on target hosts:
  - `perftest` suite (ib_send_bw, etc.) for InfiniBand testing
  - `iperf3` for TCP/UDP network testing

### Installation
```bash
git clone <repository-url>
cd tester
go mod tidy
go build -o perf-runner
```

### Basic Usage
```bash
# Use example configurations
./perf-runner -config examples/ib_send_bw-config.yaml    # InfiniBand testing
./perf-runner -config examples/iperf3-config.yaml       # TCP/UDP testing

# JSON output and verbose logging
./perf-runner -json -verbose -config mytest.yaml
```

> **For detailed configuration examples and troubleshooting, see the [User Guide](docs/USER_GUIDE.md)**

## Supported Tools

| Tool | Description | Use Case |
|------|-------------|----------|
| `ib_send_bw` | InfiniBand send bandwidth test | High-performance InfiniBand send testing |
| `iperf3` | TCP/UDP network bandwidth test | General network performance testing |

> **For detailed parameter documentation, see [Tool Parameters](docs/RUNNER_PARAMETERS.md)**

## Architecture

The tool follows a modular architecture with clear separation of concerns:

- **CLI**: Command line interface and application orchestration
- **Config**: YAML configuration loading and validation
- **Coordinator**: Test execution orchestration across hosts
- **Runner**: Abstraction for different test tools with auto-registration
- **SSH**: Remote connection and command execution
- **Output**: Result formatting and display

> **For detailed architecture information, see [Developer Guide](docs/DEVELOPER_GUIDE.md)**

## Contributing

See the [Developer Guide](docs/DEVELOPER_GUIDE.md) for development setup, architecture details, and contribution guidelines.

## Support

For issues and questions:
- Check the [User Guide](docs/USER_GUIDE.md) for troubleshooting
- Review existing issues in the repository  
- Create a new issue with detailed information
- Include configuration and log output for debugging