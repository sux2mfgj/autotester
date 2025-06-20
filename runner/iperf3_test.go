package runner

import (
	"strings"
	"testing"
	"time"
)

func TestIperf3Runner_Name(t *testing.T) {
	runner := NewIperf3Runner("")
	
	if name := runner.Name(); name != "iperf3" {
		t.Errorf("Expected name 'iperf3', got %q", name)
	}
}

func TestIperf3Runner_SupportsRole(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		role     string
		expected bool
	}{
		{"client", true},
		{"server", true},
		{"invalid", false},
		{"", false},
		{"CLIENT", false}, // case sensitive
		{"SERVER", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := runner.SupportsRole(tt.role)
			if result != tt.expected {
				t.Errorf("SupportsRole(%q) = %v, expected %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestIperf3Runner_Validate(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid server config",
			config: Config{
				Role: "server",
				Port: 5201,
			},
			wantErr: false,
		},
		{
			name: "valid client config with host",
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Port: 5201,
			},
			wantErr: false,
		},
		{
			name: "valid client config with target_host",
			config: Config{
				Role:       "client",
				TargetHost: "10.0.0.100",
				Port:       5201,
			},
			wantErr: false,
		},
		{
			name: "invalid role",
			config: Config{
				Role: "invalid",
			},
			wantErr: true,
			errMsg:  "unsupported role: invalid",
		},
		{
			name: "client without host or target_host",
			config: Config{
				Role: "client",
				Port: 5201,
			},
			wantErr: true,
			errMsg:  "target_host or host is required for client role",
		},
		{
			name: "invalid parallel streams",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"parallel_streams": 0,
				},
			},
			wantErr: true,
			errMsg:  "parallel_streams must be greater than 0",
		},
		{
			name: "invalid port number",
			config: Config{
				Role: "server",
				Port: 70000,
			},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name: "valid parallel streams",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"parallel_streams": 4,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.Validate(tt.config)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIperf3Runner_BuildCommand(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		name     string
		config   Config
		expected []string // expected flags and values
		notExpected []string // flags that should not be present
	}{
		{
			name: "basic server config",
			config: Config{
				Role: "server",
				Port: 5201,
			},
			expected: []string{"-s", "-p 5201", "-J"},
		},
		{
			name: "basic client config",
			config: Config{
				Role:       "client",
				TargetHost: "10.0.0.100",
				Port:       5201,
			},
			expected: []string{"-c 10.0.0.100", "-p 5201", "-J"},
		},
		{
			name: "client with host fallback",
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Port: 5201,
			},
			expected: []string{"-c 192.168.1.100", "-p 5201", "-J"},
		},
		{
			name: "comprehensive client config",
			config: Config{
				Role:     "client",
				Host:     "192.168.1.100",
				Port:     5555,
				Duration: 30 * time.Second,
				Args: map[string]interface{}{
					"parallel_streams": 4,
					"window_size":      "2M",
					"reverse":          true,
					"bitrate":          "1G",
					"interval":         5,
					"protocol":         "udp",
					"ipv6":             true,
					"bind_address":     "10.0.0.1",
					"omit_seconds":     3,
					"buffer_length":    "128K",
					"verbose":          true,
				},
			},
			expected: []string{
				"-c 192.168.1.100",
				"-p 5555",
				"-t 30",
				"-J",
				"-P 4",
				"-w 2M",
				"-R",
				"-b 1G",
				"-i 5",
				"-u",
				"-6",
				"-B 10.0.0.1",
				"-O 3",
				"-l 128K",
				"-V",
			},
		},
		{
			name: "server with minimal config",
			config: Config{
				Role: "server",
			},
			expected:    []string{"-s", "-J"},
			notExpected: []string{"-p", "-c"},
		},
		{
			name: "boolean flags disabled",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"reverse": false,
					"ipv6":    false,
					"verbose": false,
				},
			},
			expected:    []string{"-s", "-J"},
			notExpected: []string{"-R", "-6", "-V"},
		},
		{
			name: "ipv4 flag",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"ipv4": true,
				},
			},
			expected: []string{"-s", "-J", "-4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := runner.BuildCommand(tt.config)
			
			// Check that all expected flags are present
			for _, expected := range tt.expected {
				if !strings.Contains(cmd, expected) {
					t.Errorf("Expected %q not found in command: %s", expected, cmd)
				}
			}
			
			// Check that unwanted flags are not present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(cmd, notExpected) {
					t.Errorf("Unexpected %q found in command: %s", notExpected, cmd)
				}
			}
			
			// Verify command starts with iperf3
			if !strings.HasPrefix(cmd, "iperf3") {
				t.Errorf("Command should start with 'iperf3', got: %s", cmd)
			}
		})
	}
}

func TestIperf3Runner_ParseMetrics_JSON(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		name           string
		output         string
		expectedMetrics map[string]interface{}
	}{
		{
			name: "JSON output with bandwidth",
			output: `{
				"start": {"connected": []},
				"intervals": [],
				"end": {
					"sum_sent": {"bits_per_second": 1234567890},
					"sum_received": {"bits_per_second": 987654321},
					"streams": 4
				}
			}`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_bps":  1234567890.0,  // First found value is sum_sent
				"bandwidth_mbps": 1234.56789,
				"bandwidth_gbps": 1.23456789,
				"parallel_streams": 4,
			},
		},
		{
			name: "JSON output with retransmits",
			output: `{
				"start": {},
				"end": {
					"sum_sent": {
						"bits_per_second": 5000000000,
						"retransmits": 42
					}
				}
			}`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_bps":  5000000000.0,
				"bandwidth_mbps": 5000.0,
				"bandwidth_gbps": 5.0,
				"retransmits":    42,
			},
		},
		{
			name: "empty JSON",
			output: `{
				"start": {},
				"end": {}
			}`,
			expectedMetrics: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Output:  tt.output,
				Metrics: make(map[string]interface{}),
			}

			err := runner.ParseMetrics(result)
			if err != nil {
				t.Errorf("ParseMetrics should not return error: %v", err)
			}

			// Check all expected metrics are present and correct
			for key, expectedValue := range tt.expectedMetrics {
				actualValue, exists := result.Metrics[key]
				if !exists {
					t.Errorf("Expected metric %s not found in parsed results", key)
					continue
				}

				// Handle different numeric types
				switch expectedValue := expectedValue.(type) {
				case float64:
					if actualFloat, ok := actualValue.(float64); ok {
						if actualFloat != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualFloat)
						}
					} else {
						t.Errorf("Metric %s: expected float64, got %T", key, actualValue)
					}
				case int:
					if actualInt, ok := actualValue.(int); ok {
						if actualInt != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualInt)
						}
					} else {
						t.Errorf("Metric %s: expected int, got %T", key, actualValue)
					}
				}
			}

			// Check no unexpected metrics
			for key := range result.Metrics {
				if _, expected := tt.expectedMetrics[key]; !expected {
					t.Errorf("Unexpected metric found: %s = %v", key, result.Metrics[key])
				}
			}
		})
	}
}

func TestIperf3Runner_ParseMetrics_Text(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		name           string
		output         string
		expectedMetrics map[string]interface{}
	}{
		{
			name: "text output with Mbits/sec",
			output: `Connecting to host 192.168.1.100, port 5201
[  5] local 192.168.1.101 port 54321 connected to 192.168.1.100 port 5201
[ ID] Interval           Transfer     Bitrate         Retr  Cwnd
[  5]   0.00-10.00  sec  1.09 GBytes   934 Mbits/sec    0   85.3 KBytes
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate         Retr
[  5]   0.00-10.00  sec  1.09 GBytes   934 Mbits/sec    0             sender
[  5]   0.00-10.00  sec  1.09 GBytes   932 Mbits/sec                  receiver`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_mbps": 932.0,  // Last bandwidth value found (receiver)
				"bandwidth_bps":  932000000.0,
				"bandwidth_gbps": 0.932,
			},
		},
		{
			name: "text output with Gbits/sec",
			output: `[ ID] Interval           Transfer     Bitrate
[  5]   0.00-10.00  sec  12.5 GBytes  10.7 Gbits/sec                  receiver`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_gbps": 10.7,
				"bandwidth_bps":  10700000000.0,
				"bandwidth_mbps": 10700.0,
			},
		},
		{
			name: "text output with retransmits",
			output: `[ ID] Interval           Transfer     Bitrate         Retr  Cwnd
[  5]   0.00-10.00  sec  1.09 GBytes   934 Mbits/sec   15   85.3 KBytes`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_mbps": 934.0,
				"bandwidth_bps":  934000000.0,
				"bandwidth_gbps": 0.934,
				"retransmits":    15,
			},
		},
		{
			name: "no recognizable metrics",
			output: `iperf3: error - unable to connect to server: Connection refused`,
			expectedMetrics: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Output:  tt.output,
				Metrics: make(map[string]interface{}),
			}

			err := runner.ParseMetrics(result)
			if err != nil {
				t.Errorf("ParseMetrics should not return error: %v", err)
			}

			// Check all expected metrics are present and correct
			for key, expectedValue := range tt.expectedMetrics {
				actualValue, exists := result.Metrics[key]
				if !exists {
					t.Errorf("Expected metric %s not found in parsed results", key)
					continue
				}

				switch expectedValue := expectedValue.(type) {
				case float64:
					if actualFloat, ok := actualValue.(float64); ok {
						if actualFloat != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualFloat)
						}
					} else {
						t.Errorf("Metric %s: expected float64, got %T", key, actualValue)
					}
				case int:
					if actualInt, ok := actualValue.(int); ok {
						if actualInt != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualInt)
						}
					} else {
						t.Errorf("Metric %s: expected int, got %T", key, actualValue)
					}
				}
			}
		})
	}
}

func TestIperf3Runner_ParseMetrics_ErrorCases(t *testing.T) {
	runner := NewIperf3Runner("")

	tests := []struct {
		name        string
		result      *Result
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil result",
			result:      nil,
			expectError: true,
			errorMsg:    "result cannot be nil",
		},
		{
			name: "nil metrics map gets initialized",
			result: &Result{
				Output:  "test output",
				Metrics: nil,
			},
			expectError: false,
		},
		{
			name: "empty output",
			result: &Result{
				Output:  "",
				Metrics: make(map[string]interface{}),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.ParseMetrics(tt.result)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// Verify metrics map was initialized
				if tt.result != nil && tt.result.Metrics == nil {
					t.Error("Metrics map should be initialized")
				}
			}
		})
	}
}

func TestIperf3Runner_CustomExecutablePath(t *testing.T) {
	customPath := "/usr/local/bin/iperf3"
	runner := NewIperf3Runner(customPath)

	config := Config{
		Role: "server",
	}

	cmd := runner.BuildCommand(config)
	
	if !strings.HasPrefix(cmd, customPath) {
		t.Errorf("Expected command to start with custom path %q, got: %s", customPath, cmd)
	}
}

func TestIperf3Runner_AutoRegistration(t *testing.T) {
	// Test that iperf3 is automatically registered
	availableRunners := GetRegistered()
	
	found := false
	for _, name := range availableRunners {
		if name == "iperf3" {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("iperf3 runner should be auto-registered, available runners: %v", availableRunners)
	}
	
	// Test creating runner from registry
	runnerInstance, err := Create("iperf3")
	if err != nil {
		t.Fatalf("Failed to create iperf3 runner: %v", err)
	}
	
	if runnerInstance.Name() != "iperf3" {
		t.Errorf("Expected runner name 'iperf3', got: %s", runnerInstance.Name())
	}
}