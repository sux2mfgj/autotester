package runner

import (
	"strings"
	"testing"
)

func TestTestpmdRunner_Name(t *testing.T) {
	runner := NewTestpmdRunner("")
	if runner.Name() != "testpmd" {
		t.Errorf("Expected name 'testpmd', got %s", runner.Name())
	}
}

func TestTestpmdRunner_SupportsRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"client", "client", true},
		{"server", "server", true},
		{"intermediate", "intermediate", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	runner := NewTestpmdRunner("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.SupportsRole(tt.role)
			if result != tt.expected {
				t.Errorf("SupportsRole(%s) = %v, expected %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestTestpmdRunner_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid intermediate config",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores":           4,
					"memory_channels": 4,
					"forward_mode":    "io",
					"ports":           "0,1",
				},
			},
			shouldError: false,
		},
		{
			name: "invalid role",
			config: Config{
				Role: "invalid_role",
			},
			shouldError: true,
			errorMsg:    "unsupported role",
		},
		{
			name: "invalid cores count",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores": -1,
				},
			},
			shouldError: true,
			errorMsg:    "cores must be greater than 0",
		},
		{
			name: "invalid memory channels",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"memory_channels": 0,
				},
			},
			shouldError: true,
			errorMsg:    "memory_channels must be between 1 and 8",
		},
		{
			name: "invalid forward mode",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"forward_mode": "invalid_mode",
				},
			},
			shouldError: true,
			errorMsg:    "invalid forward_mode",
		},
		{
			name: "valid forward modes",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"forward_mode": "mac",
				},
			},
			shouldError: false,
		},
		{
			name: "role-specific args validation",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores": 2,
				},
				ServerArgs: map[string]interface{}{
					"cores": -1, // Should be ignored for intermediate role
				},
				ClientArgs: map[string]interface{}{
					"memory_channels": 9, // Should be ignored for intermediate role
				},
			},
			shouldError: false, // Only intermediate args should be validated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewTestpmdRunner("")
			err := runner.Validate(tt.config)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestTestpmdRunner_BuildCommand(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedArgs   []string
		unexpectedArgs []string
	}{
		{
			name: "basic intermediate config",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores":           "0-3",
					"memory_channels": 4,
					"ports":           "0,1",
					"forward_mode":    "io",
				},
			},
			expectedArgs: []string{
				"dpdk-testpmd", "-l 0-3", "-n 4", "--", "-i", "--portlist=0,1", "--forward-mode=io",
			},
		},
		{
			name: "comprehensive configuration",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores":           4,
					"memory_channels": 4,
					"hugepage_dir":    "/mnt/huge",
					"file_prefix":     "testpmd",
					"allow_pci":       []interface{}{"0000:01:00.0", "0000:01:00.1"},
					"ports":           "0,1",
					"rx_queues":       2,
					"tx_queues":       2,
					"rx_descriptors":  1024,
					"tx_descriptors":  1024,
					"burst_size":      32,
					"forward_mode":    "mac",
					"auto_start":      true,
					"stats_period":    5,
				},
			},
			expectedArgs: []string{
				"dpdk-testpmd", "-l 0,1,2,3", "-n 4", "--huge-dir /mnt/huge", "--file-prefix testpmd",
				"-a 0000:01:00.0", "-a 0000:01:00.1", "--", "-i", "--portlist=0,1",
				"--rxq=2", "--txq=2", "--rxd=1024", "--txd=1024", "--burst=32",
				"--forward-mode=mac", "--auto-start", "--stats-period=5",
			},
		},
		{
			name: "virtual device configuration",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores":      "0-1",
					"vdev":       []interface{}{"net_tap0,iface=tap0", "net_tap1,iface=tap1"},
					"ports":      "0,1",
					"auto_start": true,
				},
			},
			expectedArgs: []string{
				"dpdk-testpmd", "-l 0-1", "--vdev net_tap0,iface=tap0", "--vdev net_tap1,iface=tap1",
				"--", "-i", "--portlist=0,1", "--auto-start",
			},
		},
		{
			name: "client role (packet generator)",
			config: Config{
				Role: "client",
				Args: map[string]interface{}{
					"cores":        "0-3",
					"allow_pci":    "0000:01:00.0",
					"ports":        "0",
					"forward_mode": "flowgen",
					"interactive":  false,
				},
			},
			expectedArgs: []string{
				"dpdk-testpmd", "-l 0-3", "-a 0000:01:00.0", "--", "--portlist=0", "--forward-mode=flowgen",
			},
			unexpectedArgs: []string{"-i"}, // Should not be interactive for client
		},
		{
			name: "role-specific arguments",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores":        "0-1",
					"ports":        "0,1",
					"forward_mode": "io",
				},
				ServerArgs: map[string]interface{}{
					"cores": "0-7", // Should be ignored for intermediate
				},
				ClientArgs: map[string]interface{}{
					"forward_mode": "flowgen", // Should be ignored for intermediate
				},
			},
			expectedArgs: []string{
				"dpdk-testpmd", "-l 0-1", "--", "-i", "--portlist=0,1", "--forward-mode=io",
			},
			unexpectedArgs: []string{"-l 0-7", "--forward-mode=flowgen"},
		},
		{
			name: "with environment variables",
			config: Config{
				Role: "intermediate",
				Args: map[string]interface{}{
					"cores": "0-1",
					"ports": "0,1",
				},
				Env: map[string]string{
					"RTE_SDK":      "/opt/dpdk",
					"DPDK_LOG_LEVEL": "debug",
				},
			},
			expectedArgs: []string{
				"DPDK_LOG_LEVEL=debug", "RTE_SDK=/opt/dpdk", "dpdk-testpmd", "-l 0-1", "--", "-i", "--portlist=0,1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewTestpmdRunner("")
			result := runner.BuildCommand(tt.config)
			
			// Check that all expected arguments are present
			for _, expectedArg := range tt.expectedArgs {
				if !strings.Contains(result, expectedArg) {
					t.Errorf("Expected argument %q not found in command: %s", expectedArg, result)
				}
			}
			
			// Check that unexpected arguments are not present
			for _, unexpectedArg := range tt.unexpectedArgs {
				if strings.Contains(result, unexpectedArg) {
					t.Errorf("Unexpected argument %q found in command: %s", unexpectedArg, result)
				}
			}
		})
	}
}

func TestTestpmdRunner_ParseMetrics(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string]interface{}
	}{
		{
			name: "basic statistics output",
			output: `
Statistics for port 0:
RX-packets: 1000000  RX-errors: 0  RX-bytes: 64000000
TX-packets: 1000000  TX-errors: 0  TX-bytes: 64000000
Throughput: 12.5 Mpps
`,
			expected: map[string]interface{}{
				"rx_packets":       int64(1000000),
				"tx_packets":       int64(1000000),
				"rx_errors":        int64(0),
				"tx_errors":        int64(0),
				"rx_bytes":         int64(64000000),
				"tx_bytes":         int64(64000000),
				"throughput_mpps":  12.5,
				"throughput_pps":   12.5e6,
			},
		},
		{
			name: "throughput in different units",
			output: `
Throughput: 1.25 Gbps
Packet rate: 2.5 Mpps
`,
			expected: map[string]interface{}{
				"throughput_gbps":  1.25,
				"throughput_bps":   1.25e9,
				"throughput_mpps":  2.5,
				"throughput_pps":   2.5e6,
			},
		},
		{
			name: "statistics with errors",
			output: `
Statistics for port 0:
RX-packets: 999950  RX-errors: 50  RX-bytes: 63996800
TX-packets: 1000000  TX-errors: 5  TX-bytes: 64000000
`,
			expected: map[string]interface{}{
				"rx_packets": int64(999950),
				"tx_packets": int64(1000000),
				"rx_errors":  int64(50),
				"tx_errors":  int64(5),
				"rx_bytes":   int64(63996800),
				"tx_bytes":   int64(64000000),
			},
		},
		{
			name:     "empty output",
			output:   "",
			expected: map[string]interface{}{},
		},
		{
			name: "no recognizable metrics",
			output: `
testpmd> start
testpmd> stop
testpmd> quit
`,
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewTestpmdRunner("")
			result := &Result{
				Output:  tt.output,
				Metrics: make(map[string]interface{}),
			}
			
			err := runner.ParseMetrics(result)
			if err != nil {
				t.Errorf("ParseMetrics() error = %v", err)
				return
			}
			
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result.Metrics[key]; !exists {
					t.Errorf("Expected metric %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("For metric %s, expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestTestpmdRunner_ParseMetrics_NilResult(t *testing.T) {
	runner := NewTestpmdRunner("")
	err := runner.ParseMetrics(nil)
	if err == nil {
		t.Error("Expected error for nil result, got nil")
	}
}

func TestTestpmdRunner_SetExecutablePath(t *testing.T) {
	runner := NewTestpmdRunner("")
	customPath := "/custom/path/to/testpmd"
	runner.SetExecutablePath(customPath)
	
	config := Config{
		Role: "intermediate",
		Args: map[string]interface{}{
			"cores": "0-1",
		},
	}
	
	command := runner.BuildCommand(config)
	if !strings.Contains(command, customPath) {
		t.Errorf("Expected custom path %s in command, got: %s", customPath, command)
	}
}

func TestTestpmdRunner_CustomExecutablePath(t *testing.T) {
	customPath := "/opt/dpdk/bin/dpdk-testpmd"
	runner := NewTestpmdRunner(customPath)
	
	config := Config{
		Role: "intermediate",
		Args: map[string]interface{}{
			"cores": "0-1",
		},
	}
	
	command := runner.BuildCommand(config)
	if !strings.Contains(command, customPath) {
		t.Errorf("Expected custom path %s in command, got: %s", customPath, command)
	}
}

func TestTestpmdRunner_AutoRegistration(t *testing.T) {
	registered := GetRegistered()
	found := false
	for _, name := range registered {
		if name == "testpmd" {
			found = true
			break
		}
	}
	if !found {
		t.Error("testpmd runner not found in registered runners")
	}
	
	// Test creating runner through registry
	runner, err := Create("testpmd")
	if err != nil {
		t.Errorf("Failed to create testpmd runner: %v", err)
	}
	if runner.Name() != "testpmd" {
		t.Errorf("Expected runner name 'testpmd', got %s", runner.Name())
	}
}

func TestTestpmdRunner_RoleSpecificArgs(t *testing.T) {
	runner := NewTestpmdRunner("")
	config := Config{
		Role: "intermediate",
		Args: map[string]interface{}{
			"cores":        "0-1",
			"ports":        "0,1",
			"forward_mode": "io",
		},
		ServerArgs: map[string]interface{}{
			"cores":        "0-7",        // Should be ignored
			"forward_mode": "flowgen",    // Should be ignored
		},
		ClientArgs: map[string]interface{}{
			"burst_size": 64,             // Should be ignored
		},
	}
	
	command := runner.BuildCommand(config)
	
	// Should contain intermediate args
	if !strings.Contains(command, "-l 0-1") {
		t.Error("Expected intermediate cores argument not found")
	}
	if !strings.Contains(command, "--forward-mode=io") {
		t.Error("Expected intermediate forward mode not found")
	}
	
	// Should NOT contain server/client args
	if strings.Contains(command, "-l 0-7") {
		t.Error("Server args should not be used for intermediate role")
	}
	if strings.Contains(command, "--forward-mode=flowgen") {
		t.Error("Server forward mode should not be used for intermediate role")
	}
	if strings.Contains(command, "--burst=64") {
		t.Error("Client args should not be used for intermediate role")
	}
}

func TestTestpmdRunner_ValidationWithEffectiveArgs(t *testing.T) {
	runner := NewTestpmdRunner("")
	config := Config{
		Role: "intermediate",
		Args: map[string]interface{}{
			"cores":        4,
			"forward_mode": "io",
		},
		ServerArgs: map[string]interface{}{
			"cores": -1, // Invalid, but should be ignored for intermediate
		},
		ClientArgs: map[string]interface{}{
			"forward_mode": "invalid_mode", // Invalid, but should be ignored for intermediate
		},
	}
	
	// Should validate without error since only intermediate args are considered
	err := runner.Validate(config)
	if err != nil {
		t.Errorf("Validation should pass for intermediate role, got error: %v", err)
	}
}