package runner

import (
	"strings"
	"testing"
	"time"
)

func TestGetEffectiveArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected map[string]interface{}
	}{
		{
			name: "server role with server-specific args",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"common_arg": "common_value",
					"size":       1024,
				},
				ServerArgs: map[string]interface{}{
					"size":       2048, // Should override common
					"server_arg": "server_value",
				},
			},
			expected: map[string]interface{}{
				"common_arg": "common_value",
				"size":       2048, // Overridden by server_args
				"server_arg": "server_value",
			},
		},
		{
			name: "client role with client-specific args",
			config: Config{
				Role: "client",
				Args: map[string]interface{}{
					"common_arg": "common_value",
					"size":       1024,
				},
				ClientArgs: map[string]interface{}{
					"size":       4096, // Should override common
					"client_arg": "client_value",
				},
			},
			expected: map[string]interface{}{
				"common_arg": "common_value",
				"size":       4096, // Overridden by client_args
				"client_arg": "client_value",
			},
		},
		{
			name: "server role without server-specific args",
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"common_arg": "common_value",
					"size":       1024,
				},
				// No ServerArgs specified
			},
			expected: map[string]interface{}{
				"common_arg": "common_value",
				"size":       1024,
			},
		},
		{
			name: "client role ignores server args",
			config: Config{
				Role: "client",
				Args: map[string]interface{}{
					"common_arg": "common_value",
				},
				ServerArgs: map[string]interface{}{
					"server_arg": "should_be_ignored",
				},
				ClientArgs: map[string]interface{}{
					"client_arg": "client_value",
				},
			},
			expected: map[string]interface{}{
				"common_arg": "common_value",
				"client_arg": "client_value",
			},
		},
		{
			name: "empty config",
			config: Config{
				Role: "server",
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEffectiveArgs()
			
			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %s not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %s, expected %v, got %v", key, expectedValue, actualValue)
				}
			}
			
			// Check that no unexpected keys are present
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("Unexpected key %s found in result with value %v", key, result[key])
				}
			}
		})
	}
}

func TestGetEffectiveEnv(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected map[string]string
	}{
		{
			name: "server role with server-specific env",
			config: Config{
				Role: "server",
				Env: map[string]string{
					"COMMON_VAR": "common_value",
					"LD_LIBRARY_PATH": "/usr/lib",
				},
				ServerEnv: map[string]string{
					"LD_LIBRARY_PATH": "/opt/server/lib", // Should override common
					"SERVER_VAR":      "server_value",
				},
			},
			expected: map[string]string{
				"COMMON_VAR":      "common_value",
				"LD_LIBRARY_PATH": "/opt/server/lib", // Overridden by server_env
				"SERVER_VAR":      "server_value",
			},
		},
		{
			name: "client role with client-specific env",
			config: Config{
				Role: "client",
				Env: map[string]string{
					"COMMON_VAR": "common_value",
					"DEBUG":      "0",
				},
				ClientEnv: map[string]string{
					"DEBUG":      "1", // Should override common
					"CLIENT_VAR": "client_value",
				},
			},
			expected: map[string]string{
				"COMMON_VAR": "common_value",
				"DEBUG":      "1", // Overridden by client_env
				"CLIENT_VAR": "client_value",
			},
		},
		{
			name: "client role ignores server env",
			config: Config{
				Role: "client",
				Env: map[string]string{
					"COMMON_VAR": "common_value",
				},
				ServerEnv: map[string]string{
					"SERVER_VAR": "should_be_ignored",
				},
				ClientEnv: map[string]string{
					"CLIENT_VAR": "client_value",
				},
			},
			expected: map[string]string{
				"COMMON_VAR": "common_value",
				"CLIENT_VAR": "client_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEffectiveEnv()
			
			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %s not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %s, expected %v, got %v", key, expectedValue, actualValue)
				}
			}
			
			// Check that no unexpected keys are present
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("Unexpected key %s found in result with value %v", key, result[key])
				}
			}
		})
	}
}

func TestRoleSpecificArgsIntegration(t *testing.T) {
	tests := []struct {
		name           string
		runner         Runner
		config         Config
		expectedArgs   []string // Arguments we expect to be present
		unexpectedArgs []string // Arguments we expect NOT to be present
	}{
		{
			name:   "iperf3 server with role-specific args",
			runner: NewIperf3Runner(""),
			config: Config{
				Role: "server",
				Args: map[string]interface{}{
					"verbose": true,
					"interval": 1,
				},
				ServerArgs: map[string]interface{}{
					"bind_address": "0.0.0.0",
					"window_size":  "2M",
				},
				Env: map[string]string{
					"LANG": "en_US.UTF-8",
				},
				ServerEnv: map[string]string{
					"IPERF3_DEBUG": "1",
				},
			},
			expectedArgs: []string{
				"IPERF3_DEBUG=1", "LANG=en_US.UTF-8", "iperf3", "-s", "-J", "-i 1", "-V", "-B 0.0.0.0", "-w 2M",
			},
		},
		{
			name:   "iperf3 client with role-specific args",
			runner: NewIperf3Runner(""),
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Args: map[string]interface{}{
					"verbose": true,
					"window_size": "1M", // Should be overridden
				},
				ClientArgs: map[string]interface{}{
					"parallel_streams": 4,
					"window_size":      "128K", // Override common window_size
				},
				Env: map[string]string{
					"LANG": "en_US.UTF-8",
				},
				ClientEnv: map[string]string{
					"IPERF3_CLIENT_DEBUG": "1",
				},
			},
			expectedArgs: []string{
				"IPERF3_CLIENT_DEBUG=1", "LANG=en_US.UTF-8", "iperf3", "-c 192.168.1.100", "-J", "-V", "-P 4", "-w 128K",
			},
			unexpectedArgs: []string{"-w 1M"}, // Should not contain the overridden window size
		},
		{
			name:   "ib_send_bw with role-specific queue depths",
			runner: NewIbSendBwRunner(""),
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Args: map[string]interface{}{
					"size":       32768,
					"iterations": 1000,
					"connection": "RC",
				},
				ClientArgs: map[string]interface{}{
					"tx_depth": 512,
					"rx_depth": 64,
				},
				ServerArgs: map[string]interface{}{
					"tx_depth": 64,  // Should be ignored for client
					"rx_depth": 512, // Should be ignored for client
				},
			},
			expectedArgs: []string{
				"ib_send_bw", "192.168.1.100", "-s 32768", "-n 1000", "-c RC", "-t 512", "-r 64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.runner.BuildCommand(tt.config)
			
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

func TestRoleSpecificArgsValidation(t *testing.T) {
	config := Config{
		Role:     "client",
		Duration: 30 * time.Second,
		Args: map[string]interface{}{
			"parallel_streams": 4,
		},
		ClientArgs: map[string]interface{}{
			"parallel_streams": 8, // Should override and be validated
		},
		Host: "192.168.1.100",
	}

	runner := NewIperf3Runner("")
	
	// Test that validation uses effective args
	err := runner.Validate(config)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	
	// Test with invalid role-specific args
	config.ClientArgs["parallel_streams"] = -1 // Invalid value
	err = runner.Validate(config)
	if err == nil {
		t.Error("Expected validation to fail with negative parallel_streams")
	}
}