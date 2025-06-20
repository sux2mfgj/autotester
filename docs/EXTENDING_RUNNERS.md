# Extending Runners - Adding New Test Tools

This detailed technical guide walks you through adding support for new test tools to the performance testing framework. This document is intended for developers who want to extend the system with new runners.

> **For general development information, see [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)**
> 
> **For user documentation, see [USER_GUIDE.md](USER_GUIDE.md)**

## Overview

The tester uses a self-contained, modular architecture where each test tool is implemented as a `Runner` that auto-registers itself. To add a new command, you only need to:

1. Create the runner file (auto-registering)
2. Update configuration examples  
3. Document parameters in [RUNNER_PARAMETERS.md](RUNNER_PARAMETERS.md)

**Key Benefits:**
- **High Modularity**: Runners are completely self-contained
- **Auto-Registration**: No need to modify CLI or coordinator code
- **Single Responsibility**: Each runner handles its own command building
- **Fewer File Changes**: Adding a runner touches minimal files

## Step-by-Step Guide

### Step 1: Create Self-Contained Runner

Create a new file in the `runner/` package for your tool. For example, let's add support for `ib_read_bw`. The runner must implement the `Runner` interface and auto-register itself:

```go
// runner/ib_read_bw.go
package runner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Auto-register the ib_read_bw runner
func init() {
	Register("ib_read_bw", func() Runner {
		return NewIbReadBwRunner("")
	})
}

// IbReadBwRunner implements the Runner interface for ib_read_bw
type IbReadBwRunner struct {
	executablePath string
}

// NewIbReadBwRunner creates a new ib_read_bw runner
func NewIbReadBwRunner(executablePath string) *IbReadBwRunner {
	if executablePath == "" {
		executablePath = "ib_read_bw"
	}
	return &IbReadBwRunner{
		executablePath: executablePath,
	}
}

// Name returns the name of the runner
func (r *IbReadBwRunner) Name() string {
	return "ib_read_bw"
}

// SupportsRole returns true if the runner supports the given role
func (r *IbReadBwRunner) SupportsRole(role string) bool {
	return role == "client" || role == "server"
}

// Validate checks if the configuration is valid for ib_read_bw
func (r *IbReadBwRunner) Validate(config Config) error {
	if !r.SupportsRole(config.Role) {
		return fmt.Errorf("unsupported role: %s", config.Role)
	}
	
	if config.Role == "client" && config.Host == "" {
		return fmt.Errorf("host is required for client role")
	}
	
	return nil
}


// BuildCommand constructs the full command line for remote execution
func (r *IbReadBwRunner) BuildCommand(config Config) string {
	cmd := r.executablePath
	
	// Client mode needs a host argument, server mode doesn't
	if config.Role == "client" && config.Host != "" {
		cmd += fmt.Sprintf(" %s", config.Host)
	}
	
	// Port (if specified)
	if config.Port > 0 {
		cmd += fmt.Sprintf(" -p %d", config.Port)
	}
	
	// Duration (if specified)
	if config.Duration > 0 {
		cmd += fmt.Sprintf(" -D %d", int(config.Duration.Seconds()))
	}
	
	// Additional arguments from config
	for key, value := range config.Args {
		switch key {
		case "size":
			cmd += fmt.Sprintf(" -s %v", value)
		case "iterations":
			cmd += fmt.Sprintf(" -n %v", value)
		case "tx_depth":
			cmd += fmt.Sprintf(" -t %v", value)
		case "rx_depth":
			cmd += fmt.Sprintf(" -r %v", value)
		case "connection":
			cmd += fmt.Sprintf(" -c %v", value)
		case "bidirectional":
			if bidir, ok := value.(bool); ok && bidir {
				cmd += " -b"
			}
		}
	}
	
	return cmd
}


// ParseMetrics extracts performance metrics from ib_read_bw output
func (r *IbReadBwRunner) ParseMetrics(result *Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	
	if result.Metrics == nil {
		result.Metrics = make(map[string]interface{})
	}
	output := result.Output
	
	// Parse bandwidth (similar format to ib_send_bw)
	// Example: "1000.00 MB/sec" or "8.00 Gb/sec"
	bandwidthRegex := regexp.MustCompile(`(\d+\.?\d*)\s*(MB/sec|Gb/sec|GB/sec)`)
	if matches := bandwidthRegex.FindStringSubmatch(output); len(matches) >= 3 {
		if bw, err := strconv.ParseFloat(matches[1], 64); err == nil {
			unit := matches[2]
			switch unit {
			case "MB/sec":
				result.Metrics["bandwidth_mbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e6 * 8
			case "GB/sec":
				result.Metrics["bandwidth_gbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e9 * 8
			case "Gb/sec":
				result.Metrics["bandwidth_gbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e9
			}
			result.Metrics["bandwidth_readable"] = matches[0]
		}
	}
	
	// Parse connection information
	if strings.Contains(output, "Connection type:") {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Connection type:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					result.Metrics["connection_type"] = strings.TrimSpace(parts[1])
				}
			}
		}
	}
	
	return nil
}
```

**That's it!** The runner is now automatically available. The `init()` function registers it when the package is imported, and the CLI will automatically discover it.

### Step 2: Create Configuration Examples

Add example configurations for the new tool:

```yaml
# examples/ib_read_bw-config.yaml
name: "InfiniBand Read Bandwidth Test"
description: "Test InfiniBand read bandwidth using ib_read_bw"
runner: "ib_read_bw"
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
  - name: "IB Read BW Test"
    description: "Basic InfiniBand read bandwidth test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536      # 64KB messages
        iterations: 1000
        connection: "RC"

  - name: "IB Read BW Bidirectional"
    description: "Bidirectional read bandwidth test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 60s
      args:
        size: 131072     # 128KB messages
        iterations: 500
        bidirectional: true
        connection: "RC"
```

### Step 3: Document Parameters

Add the new tool's parameters to [RUNNER_PARAMETERS.md](RUNNER_PARAMETERS.md):

```markdown
## ib_read_bw Runner

The `ib_read_bw` runner executes InfiniBand read bandwidth tests using the perftest suite.

### ib_read_bw Arguments

| Argument | Type | Description |
|----------|------|-------------|
| `size` | int | Message size in bytes |
| `iterations` | int | Number of iterations to run |
| `tx_depth` | int | Send queue depth |
| `rx_depth` | int | Receive queue depth |
| `connection` | string | Connection type (RC/UC/UD) |
| `bidirectional` | bool | Bidirectional test |

### Configuration Examples

[Include configuration examples here]
```

### That's All!

Update the supported tools table in README.md:

```markdown
### Supported Test Tools

| Tool | Description | Use Case |
|------|-------------|----------|
| `ib_send_bw` | InfiniBand send bandwidth test | High-performance InfiniBand send testing |
| `ib_read_bw` | InfiniBand read bandwidth test | High-performance InfiniBand read testing |
```

## Advanced Features

### Custom Argument Parsing

For tools with complex argument structures, you can implement custom parsing:

```go
func (r *CustomPerfTestRunner) buildComplexArgs(config Config) []string {
	args := []string{r.executablePath}
	
	// Handle nested configurations
	if perfConfig, ok := config.Args["performance"].(map[string]interface{}); ok {
		if queueDepth, ok := perfConfig["queue_depth"].(int); ok {
			args = append(args, "-q", strconv.Itoa(queueDepth))
		}
		if mtu, ok := perfConfig["mtu"].(int); ok {
			args = append(args, "--mtu", strconv.Itoa(mtu))
		}
	}
	
	return args
}
```

### Custom Metrics Parsing

Implement sophisticated output parsing for complex perftest tools:

```go
func (r *CustomPerfTestRunner) parseAdvancedMetrics(result *Result) {
	lines := strings.Split(result.Output, "\n")
	
	for _, line := range lines {
		// Parse structured perftest output
		if strings.HasPrefix(line, "#bytes") {
			// Handle perftest table headers
			continue
		}
		
		// Parse data lines
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			if bytes, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
				result.Metrics["message_size"] = bytes
			}
			if bw, err := strconv.ParseFloat(fields[2], 64); err == nil {
				result.Metrics["bandwidth_peak"] = bw
			}
		}
	}
}
```

## Testing Your New Runner

### Unit Tests

Create unit tests for your runner:

```go
// runner/ib_read_bw_test.go
package runner

import (
	"context"
	"testing"
	"time"
)

func TestIbReadBwRunner_Validate(t *testing.T) {
	runner := NewIbReadBwRunner("")
	
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid client config",
			config: Config{
				Role: "client",
				Host: "example.com",
				Port: 18515,
			},
			wantErr: false,
		},
		{
			name: "invalid role",
			config: Config{
				Role: "invalid",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

## Common Perftest Tools to Add

Here are some common perftest suite tools you might want to add:

1. **ib_write_bw** - InfiniBand write bandwidth test
2. **ib_atomic_bw** - InfiniBand atomic operations bandwidth test
3. **ib_read_lat** - InfiniBand read latency test
4. **ib_write_lat** - InfiniBand write latency test
5. **ib_send_lat** - InfiniBand send latency test

Each follows similar patterns but may have tool-specific arguments and output formats.

## Benefits of the New Architecture

**✅ High Modularity**: Each runner is completely self-contained
**✅ Auto-Registration**: No need to modify CLI or coordinator code  
**✅ Single Responsibility**: Each runner handles its own command building
**✅ Fewer File Changes**: Adding a runner touches minimal files
**✅ Better Testing**: Test actual runners, not mocks or separate builders

## Best Practices

1. **Auto-Registration**: Always include the `init()` function with `Register()` call
2. **Complete Interface**: Implement all Runner interface methods, especially `BuildCommand`
3. **Error Handling**: Always validate configurations and provide meaningful error messages
4. **Timeouts**: Respect context cancellation and timeouts
5. **Metrics**: Extract as much useful information as possible from perftest output
6. **Testing**: Write comprehensive tests for your runner
7. **Consistency**: Follow existing patterns in the codebase

## Common Pitfalls

1. **Forgetting Auto-Registration**: Don't forget the `init()` function
2. **Command Injection**: Always validate and sanitize arguments
3. **Platform Differences**: Consider different InfiniBand hardware configurations
4. **Output Parsing**: Handle different perftest output formats and versions
5. **Resource Cleanup**: Ensure proper cleanup of processes and resources
6. **Error Propagation**: Don't swallow important error information

The streamlined, modular architecture makes it straightforward to add new perftest tools with minimal code changes and maximum maintainability.