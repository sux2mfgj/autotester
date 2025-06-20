# Perftest Tester Architecture Documentation

## Overview

The Perftest Tester is a Go-based tool for executing InfiniBand performance tests using the perftest suite (like ib_send_bw) across multiple hosts via SSH connections. It features a modular architecture with clear separation of concerns.

## System Architecture

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
               │command_build│        │
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

## Package Overview

### 1. **main** (Entry Point)
- **Purpose**: Application entry point
- **Responsibilities**: 
  - Initialize CLI application
  - Handle top-level error reporting
- **Dependencies**: `cli`

### 2. **cli** (Command Line Interface)
- **Purpose**: Handle command line arguments and application orchestration
- **Files**:
  - `flags.go`: Command line flag definitions
  - `app.go`: Main application logic and flow control

#### Dependencies:
- `config` - Load configuration
- `coordinator` - Execute tests
- `output` - Format results
- `runner` - Register runners

### 3. **config** (Configuration Management)
- **Purpose**: Handle YAML configuration loading and validation
- **Files**:
  - `config.go`: Configuration structures and loading
  - `validator.go`: Configuration validation logic

#### Key Types:
```go
type TestConfig struct {
    Name        string
    Runner      string
    Hosts       map[string]*HostConfig
    Tests       []TestScenario
}

type HostConfig struct {
    SSH      *ssh.Config
    Role     string
    Runner   *runner.Config
}

type TestScenario struct {
    Name        string
    Client      string  // Host name
    Server      string  // Host name
    Config      *runner.Config
}
```

#### Dependencies:
- `ssh` - SSH configuration
- `runner` - Runner configuration

### 4. **coordinator** (Test Orchestration)
- **Purpose**: Coordinate test execution across multiple hosts
- **Files**:
  - `coordinator.go`: Main coordinator with SSH connection management
  - `executor.go`: Individual test execution logic
  - `types.go`: Result type definitions

#### Key Types:
```go
type Coordinator struct {
    config     *config.TestConfig
    runners    map[string]runner.Runner
    sshClients map[string]*ssh.Client
}

type TestResult struct {
    ScenarioName string
    Success      bool
    ClientResult *runner.Result
    ServerResult *runner.Result
}
```

#### Dependencies:
- `config` - Test configuration
- `ssh` - Remote execution
- `runner` - Test runners

### 5. **runner** (Test Execution Abstraction)
- **Purpose**: Abstract interface for different perftest tools
- **Files**:
  - `runner.go`: Runner interface definition
  - `ib_send_bw.go`: InfiniBand send bandwidth test implementation

#### Key Interface:
```go
type Runner interface {
    Validate(config Config) error
    Name() string
    SupportsRole(role string) bool
    BuildCommand(config Config) string
    ParseMetrics(result *Result) error
}
```

#### Dependencies:
- None (core abstraction)

### 6. **ssh** (SSH Connection Management)
- **Purpose**: Handle SSH connections and remote command execution
- **Files**:
  - `client.go`: SSH client implementation

#### Key Types:
```go
type Client struct {
    config *Config
    client *ssh.Client
}

type Config struct {
    Host     string
    User     string
    KeyPath  string
    Password string
}
```

#### Dependencies:
- `golang.org/x/crypto/ssh` - SSH protocol

### 7. **output** (Result Formatting)
- **Purpose**: Format and display test results
- **Files**:
  - `formatter.go`: JSON and text output formatting

#### Key Types:
```go
type Formatter struct {
    jsonOutput bool
}
```

#### Dependencies:
- `coordinator` - TestResult types

## Data Flow

### 1. **Initialization Flow**
```
main.go → cli.NewApp() → cli.App.Run()
```

### 2. **Configuration Flow**
```
CLI → config.LoadConfig() → validator.ValidateConfig()
```

### 3. **Test Execution Flow**
```
CLI → coordinator.NewCoordinator()
    → coordinator.ConnectHosts() (SSH connections)
    → coordinator.RunAllTests()
        → executor.ExecuteTest() (for each test)
            → executor.runRemoteCommand() (client & server)
                → ssh.ExecuteCommand()
                → runner.Run() (remote execution)
```

### 4. **Output Flow**
```
coordinator.TestResults → output.Formatter → stdout (JSON/Text)
```

## Key Design Patterns

### 1. **Strategy Pattern**
- **Interface**: `runner.Runner`
- **Implementations**: `IbSendBwRunner`
- **Purpose**: Support different perftest tools

### 2. **Registry Pattern**
- **Implementation**: Auto-registration system in `runner` package
- **Purpose**: Automatically discover and register runner implementations

### 3. **Factory Pattern**
- **Usage**: Runner creation from registry
- **Purpose**: Create appropriate runner instances by name

### 4. **Template Method Pattern**
- **Implementation**: Test execution flow in executor
- **Purpose**: Standardize client-server coordination

## Extension Points

### Adding New Perftest Runners
1. Implement `runner.Runner` interface
2. Add auto-registration in `init()` function
3. The runner will be automatically discovered

### Adding New Output Formats
1. Extend `output.Formatter`
2. Add new CLI flag
3. Implement format-specific logic

### Adding New SSH Authentication
1. Extend `ssh.Config`
2. Add authentication logic in `ssh.Client.Connect()`

## Configuration Examples

### Host Configuration
```yaml
hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      port: 18515
```

### Test Scenario
```yaml
tests:
  - name: "IB Send BW Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        iterations: 1000
        connection: "RC"
```

## Error Handling Strategy

### 1. **Configuration Errors**
- Validated at startup
- Fail fast with descriptive messages

### 2. **Connection Errors**
- Retry logic in SSH client
- Graceful degradation

### 3. **Test Execution Errors**
- Individual test failures don't stop execution
- Comprehensive error reporting

## Security Considerations

### 1. **SSH Authentication**
- Key-based authentication preferred
- Host key verification (currently relaxed for demo)

### 2. **Configuration Security**
- No secrets in configuration files
- SSH key paths, not embedded keys

### 3. **Command Injection Prevention**
- Parameterized command building
- Input validation

## Performance Considerations

### 1. **Concurrent Connections**
- Parallel SSH connections to hosts
- Connection pooling and reuse

### 2. **Test Coordination**
- Server started before client
- Proper synchronization

### 3. **Resource Management**
- Graceful shutdown
- Connection cleanup

## InfiniBand Specific Features

### 1. **RDMA Operations**
- Support for different connection types (RC, UC, UD)
- Queue pair management
- Memory registration optimization

### 2. **Performance Metrics**
- Bandwidth measurements in MB/sec and Gb/sec
- Message rate calculations
- Latency analysis with histograms

### 3. **Hardware Considerations**
- MTU size configuration
- Service level settings
- GID index management

## Future Enhancement Areas

1. **Additional Perftest Tools** (ib_read_bw, ib_write_bw, ib_atomic_bw)
2. **Real-time Monitoring Dashboard**
3. **Performance Trend Analysis**
4. **Automated Tuning Recommendations**
5. **Container Support for Testing**
6. **Integration with HPC Job Schedulers**