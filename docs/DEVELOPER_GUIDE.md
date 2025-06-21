# Developer Guide - InfiniBand Performance Testing Tool

This guide provides comprehensive information for developers working on the perf-runner tool, including architecture, extending functionality, and contributing to the project.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Development Setup](#development-setup)
- [Adding New Test Tools](#adding-new-test-tools)
- [Code Structure](#code-structure)
- [Testing](#testing)
- [Contributing](#contributing)

## Architecture Overview

The perf-runner follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                           main.go                               │
│                      (Entry Point)                              │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                       CLI Package                               │
│  ┌─────────────┐  ┌──────────────────────────────────────────┐  │
│  │  flags.go   │  │              app.go                      │  │
│  │(CLI Flags)  │  │         (Application Logic)             │  │
│  └─────────────┘  └──────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
      ┌───────────────┼───────────────┐
      │               │               │
┌─────▼─────┐  ┌──────▼──────┐  ┌─────▼──────┐
│   Config  │  │ Coordinator │  │   Output   │
│  Package  │  │   Package   │  │  Package   │
└───────────┘  └─────────────┘  └────────────┘
      │               │               │
┌─────▼─────┐  ┌──────▼──────┐        │
│ validator │  │  executor   │        │
└───────────┘  │    types    │        │
               └─────────────┘        │
                      │               │
              ┌───────▼───────┐       │
              │     Runner    │       │
              │    Package    │       │
              └───────────────┘       │
                      │               │
              ┌───────▼───────┐       │
              │      SSH      │       │
              │    Package    │       │
              └───────────────┘       │
                                      │
                              ┌───────▼────┐
                              │ formatter  │
                              └────────────┘
```

### Key Design Patterns

#### 1. **Strategy Pattern**
- **Interface**: `runner.Runner`
- **Implementations**: `IbSendBwRunner`, `Iperf3Runner`
- **Purpose**: Support different test tools with consistent interface

#### 2. **Registry Pattern**
- **Implementation**: Auto-registration system in `runner` package
- **Purpose**: Automatically discover and register runner implementations

#### 3. **Factory Pattern**
- **Usage**: Runner creation from registry
- **Purpose**: Create appropriate runner instances by name

## Development Setup

### Prerequisites
- Go 1.21+
- Access to test hosts with SSH
- Test tools installed (ib_send_bw, iperf3, etc.)

### Building
```bash
git clone <repository-url>
cd tester
go mod tidy
go build -o perf-runner
```

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./runner
go test -v ./config
```

### Code Structure
```
tester/
├── main.go                 # Entry point
├── cli/                    # Command line interface
│   ├── app.go             # Application orchestration
│   └── flags.go           # CLI flag definitions
├── config/                 # Configuration management
│   ├── config.go          # Config loading and structures
│   └── validator.go       # Configuration validation
├── coordinator/            # Test orchestration
│   ├── coordinator.go     # Main coordinator
│   ├── executor.go        # Test execution logic
│   └── types.go           # Result types
├── runner/                 # Test tool abstraction
│   ├── runner.go          # Runner interface and registry
│   ├── ib_send_bw.go      # InfiniBand send bandwidth runner
│   └── iperf3.go          # TCP/UDP bandwidth runner
├── ssh/                    # SSH client implementation
│   └── client.go          # SSH connection and execution
├── output/                 # Result formatting
│   └── formatter.go       # JSON and text output
├── examples/               # Example configurations
└── docs/                   # Documentation
```

## Adding New Test Tools

The modular architecture makes it straightforward to add new test tools. Each tool is implemented as a `Runner` that auto-registers itself.

### Step-by-Step Guide

#### Step 1: Create Self-Contained Runner

Create a new file in the `runner/` package. For example, to add support for `ib_read_bw`:

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
	
	if config.Role == "client" && config.Host == "" && config.TargetHost == "" {
		return fmt.Errorf("target_host or host is required for client role")
	}
	
	return nil
}

// BuildCommand constructs the full command line for remote execution
func (r *IbReadBwRunner) BuildCommand(config Config) string {
	cmd := r.executablePath
	
	// Client mode needs a host argument, server mode doesn't
	if config.Role == "client" {
		targetHost := config.TargetHost
		if targetHost == "" {
			targetHost = config.Host
		}
		cmd += fmt.Sprintf(" %s", targetHost)
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
		// Add more parameters as needed
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
	
	return nil
}
```

**That's it!** The runner is now automatically available through the auto-registration system.

> **For a complete step-by-step example with full code, see [Extending Runners](EXTENDING_RUNNERS.md)**

#### Step 2: Create Tests

Create comprehensive tests for your runner:

```go
// runner/ib_read_bw_test.go
package runner

import (
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

// Add more tests for BuildCommand, ParseMetrics, etc.
```

#### Step 3: Create Configuration Examples

Add example configurations:

```yaml
# examples/ib_read_bw-config.yaml
name: "InfiniBand Read Bandwidth Test"
runner: "ib_read_bw"
timeout: 5m

hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"

  ib_client:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"

tests:
  - name: "IB Read BW Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        iterations: 1000
```

#### Step 4: Update Documentation

1. Add the new tool's parameters to `RUNNER_PARAMETERS.md`
2. Update the supported tools table in `README.md`
3. Add example configurations to the `examples/` directory

> **For detailed examples, see [Extending Runners](EXTENDING_RUNNERS.md)**

### Runner Interface Requirements

All runners must implement the `Runner` interface:

```go
type Runner interface {
	// Validate checks if the configuration is valid for this runner
	Validate(config Config) error
	
	// Name returns the name of the runner
	Name() string
	
	// SupportsRole returns true if the runner supports the given role
	SupportsRole(role string) bool
	
	// BuildCommand constructs the command line for remote execution
	BuildCommand(config Config) string
	
	// ParseMetrics extracts performance metrics from command output
	ParseMetrics(result *Result) error
}
```

### Best Practices

1. **Auto-Registration**: Always include the `init()` function with `Register()` call
2. **Complete Interface**: Implement all Runner interface methods
3. **Error Handling**: Always validate configurations and provide meaningful error messages
4. **Comprehensive Testing**: Write tests for all methods and edge cases
5. **Documentation**: Document all parameters and provide examples
6. **Consistency**: Follow existing patterns in the codebase

### Common Implementation Patterns

#### Parameter Handling
```go
// Handle different parameter types safely
for key, value := range config.Args {
	switch key {
	case "size":
		if size, ok := value.(int); ok {
			cmd += fmt.Sprintf(" -s %d", size)
		} else if sizeStr, ok := value.(string); ok {
			cmd += fmt.Sprintf(" -s %s", sizeStr)
		}
	case "enabled_flag":
		if enabled, ok := value.(bool); ok && enabled {
			cmd += " --flag"
		}
	}
}
```

#### Metrics Parsing
```go
// Robust metrics parsing with error handling
func (r *MyRunner) ParseMetrics(result *Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	
	if result.Metrics == nil {
		result.Metrics = make(map[string]interface{})
	}
	
	// Parse metrics with proper error handling
	lines := strings.Split(result.Output, "\n")
	for _, line := range lines {
		if matches := someRegex.FindStringSubmatch(line); len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				result.Metrics["metric_name"] = value
			}
		}
	}
	
	return nil
}
```

## Testing

### Test Structure

The project maintains high test coverage with comprehensive test suites:

```bash
# Current coverage
runner package: 96.5% of statements
config package: 76.5% of statements
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **Validation Tests**: Test configuration validation
4. **Parsing Tests**: Test output parsing with various formats
5. **Error Tests**: Test error handling and edge cases

### Writing Tests

Follow existing test patterns:

```go
func TestRunner_Method(t *testing.T) {
	tests := []struct {
		name     string
		input    InputType
		expected ExpectedType
		wantErr  bool
	}{
		{
			name:     "valid case",
			input:    validInput,
			expected: expectedOutput,
			wantErr:  false,
		},
		{
			name:    "error case",
			input:   invalidInput,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := methodUnderTest(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
```

## Contributing

### Development Workflow

1. **Fork** the repository
2. **Create** a feature branch
3. **Implement** your changes following existing patterns
4. **Add tests** with good coverage
5. **Update documentation** as needed
6. **Commit** with atomic, descriptive messages
7. **Submit** a pull request

### Code Standards

- Follow Go best practices and idioms
- Maintain high test coverage (>90%)
- Include comprehensive error handling
- Write clear, descriptive commit messages
- Update documentation for user-facing changes

### Commit Message Format

```
type: brief description

- Detailed explanation if needed
- Use bullet points for multiple changes
- Reference issues if applicable

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Pull Request Guidelines

- Ensure all tests pass
- Include test coverage for new functionality
- Update documentation for user-facing changes
- Follow atomic commit principles
- Provide clear description of changes

### Architecture Decisions

When making significant changes:

1. Consider impact on existing functionality
2. Maintain backward compatibility when possible
3. Follow established patterns and conventions
4. Consider performance implications
5. Ensure security best practices

The modular architecture makes it easy to extend functionality while maintaining stability and consistency across the codebase.