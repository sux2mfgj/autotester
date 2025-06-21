# User Guide - InfiniBand Performance Testing Tool

This guide provides comprehensive information for users running performance tests with the perf-runner tool.

## Table of Contents

- [Quick Start](#quick-start)
- [Supported Tools](#supported-tools)
- [Configuration](#configuration)
- [Running Tests](#running-tests)
- [Understanding Results](#understanding-results)
- [Troubleshooting](#troubleshooting)

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

1. **Create a configuration file** (see examples below or in `examples/` directory)
2. **Run tests**:

```bash
# Use example configuration
./perf-runner -config examples/ib_send_bw-config.yaml

# Use custom configuration
./perf-runner -config mytest.yaml

# JSON output
./perf-runner -json

# Verbose logging
./perf-runner -verbose
```

## Supported Tools

| Tool | Description | Use Case |
|------|-------------|----------|
| `ib_send_bw` | InfiniBand send bandwidth test | High-performance InfiniBand send testing |
| `iperf3` | TCP/UDP network bandwidth test | General network performance testing |

## Configuration

### Configuration File Structure

```yaml
name: "Test Name"
description: "Optional description"
runner: "tool_name"  # ib_send_bw or iperf3
timeout: 5m

hosts:
  host1:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      # host-specific parameters

  host2:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"

tests:
  - name: "Test 1"
    client: "host2"
    server: "host1"
    config:
      duration: 30s
      args:
        # test-specific parameters
```

### SSH Configuration

Each host requires SSH connection details:

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
      # parameters specific to the tool
```

### Separate Networks

You can use different networks for SSH management and testing:

```yaml
hosts:
  server:
    ssh:
      host: "192.168.1.100"    # Management network for SSH
    runner:
      # For server, no target_host needed
      
  client:
    ssh:
      host: "192.168.1.101"    # Management network for SSH  
    runner:
      target_host: "10.0.0.100"  # Test network IP
```

## Running Tests

### Command Line Options

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

### Test Execution Flow

1. **Configuration Loading**: Validates YAML configuration
2. **SSH Connections**: Establishes connections to all hosts
3. **Test Execution**: Runs tests sequentially with proper client-server coordination
4. **Results Collection**: Gathers output and parses metrics
5. **Report Generation**: Displays results in requested format

### Test Scenarios

Define multiple test scenarios with different parameters:

```yaml
tests:
  - name: "Basic Test"
    description: "Basic performance test"
    client: "client_host"
    server: "server_host"
    config:
      duration: 30s
      args:
        # test parameters
    repeat: 3                     # Run 3 times
    delay: 5s                     # 5s delay between runs
```

## Understanding Results

### Output Formats

The tool supports both human-readable text output and structured JSON output:

#### Text Output
Displays test results in a readable format with:
- Test execution status and timing
- Command lines executed on each host  
- Complete stdout/stderr output from tests
- Parsed performance metrics
- Detailed error information for failed tests

#### JSON Output
Provides structured output suitable for parsing and integration:
```bash
./tester -json -config mytest.yaml
```

### Metrics

Different tools provide different metrics:

#### InfiniBand Tools (ib_send_bw)
- `bandwidth_mbps` - Bandwidth in MB/sec
- `bandwidth_gbps` - Bandwidth in Gb/sec
- `bandwidth_bps` - Bandwidth in bits per second
- `message_rate_pps` - Message rate in packets per second
- `bytes` - Message size in bytes
- `iterations` - Number of iterations completed
- `connection_type` - Connection type used (RC/UC/UD)

#### TCP/UDP Tools (iperf3)
- `bandwidth_bps` - Bandwidth in bits per second
- `bandwidth_mbps` - Bandwidth in megabits per second
- `bandwidth_gbps` - Bandwidth in gigabits per second
- `retransmits` - TCP retransmission count
- `parallel_streams` - Number of parallel streams used
- `actual_duration` - Actual test duration

## Troubleshooting

### Common Issues

#### SSH Connection Problems
1. **Connection refused**: Ensure SSH service is running on target hosts
2. **Permission denied**: Check SSH key permissions and user access
3. **Timeout errors**: Increase timeout values in configuration

#### Tool Execution Problems
1. **Command not found**: Ensure test tools are installed and in PATH
2. **Permission denied**: Check user permissions for test tools
3. **Port conflicts**: Ensure test ports are available and not in use

#### Network Issues
1. **InfiniBand errors**: Verify IB hardware is properly configured and active
2. **RDMA device not found**: Check that InfiniBand drivers and devices are available
3. **Firewall blocking**: Check firewall rules for test ports

### Debug Mode

Run with `-verbose` flag for detailed logging:

```bash
./tester -verbose -config debug.yaml
```

### Log Analysis

Verbose output includes:
- SSH connection establishment
- Command execution on each host
- Raw tool output
- Metrics parsing details
- Error messages and stack traces

### Configuration Validation

The tool validates configuration at startup and reports:
- Missing required fields
- Invalid host references
- Malformed parameters
- SSH connection issues

### Getting Help

For additional help:
- Check example configurations in `examples/` directory
- Review tool-specific parameter documentation
- Enable verbose logging for detailed execution information
- Verify SSH connectivity manually before running tests

## Tool-Specific Information

For detailed information about specific tools, including parameters, configuration examples, and troubleshooting:

- [InfiniBand Tools (ib_send_bw)](RUNNER_PARAMETERS.md#ib_send_bw-runner)
- [TCP/UDP Tools (iperf3)](RUNNER_PARAMETERS.md#iperf3-runner)

## Examples

The `examples/` directory contains complete working configurations:
- `ib_send_bw-config.yaml` - InfiniBand testing examples
- `iperf3-config.yaml` - TCP/UDP network testing examples