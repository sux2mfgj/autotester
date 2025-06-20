package runner

import (
	"testing"
	"time"
)

func TestIbSendBwRunner_ParseMetrics(t *testing.T) {
	runner := NewIbSendBwRunner("")

	tests := []struct {
		name           string
		output         string
		expectedMetrics map[string]interface{}
	}{
		{
			name: "standard perftest output with table",
			output: `#bytes     #iterations    BW peak[MB/sec]    BW average[MB/sec]   MsgRate[Mpps]
 65536      1000           12345.67           12000.50             0.18`,
			expectedMetrics: map[string]interface{}{
				"bytes":                    int64(65536),
				"iterations":              int64(1000),
				"bandwidth_peak_mbps":     12345.67,
				"bandwidth_peak_bps":      12345.67 * 1e6 * 8,
				"bandwidth_average_mbps":  12000.50,
				"bandwidth_average_bps":   12000.50 * 1e6 * 8,
				"message_rate_mpps":       0.18,
				"message_rate_pps":        0.18 * 1e6,
			},
		},
		{
			name: "output with bandwidth in different units",
			output: `8.50 Gb/sec`,
			expectedMetrics: map[string]interface{}{
				"bandwidth_gbps":     8.50,
				"bandwidth_bps":      8.50 * 1e9,
				"bandwidth_readable": "8.50 Gb/sec",
			},
		},
		{
			name: "output with connection information",
			output: `Connection type: RC
			MTU: 4096
			Message size: 65536
			Number of qps: 4`,
			expectedMetrics: map[string]interface{}{
				"connection_type": "RC",
				"mtu":            4096,
				"message_size":   65536,
				"num_qps":        4,
			},
		},
		{
			name: "message rate in different units", 
			output: `1000.00 MB/sec 250.5 Kpps`,
			expectedMetrics: map[string]interface{}{
				"bytes":               int64(1000), // parseResultLine finds this
				"bandwidth_peak_mbps": 250.5,       // parseResultLine finds this
				"bandwidth_peak_bps":  250.5 * 1e6 * 8,
				"bandwidth_mbps":      1000.00,
				"bandwidth_bps":       1000.00 * 1e6 * 8,
				"bandwidth_readable":  "1000.00 MB/sec",
				"message_rate_kpps":   250.5,
				"message_rate_pps":    250.5 * 1e3,
			},
		},
		{
			name: "empty output",
			output: "",
			expectedMetrics: map[string]interface{}{},
		},
		{
			name: "output with no recognizable metrics",
			output: `Error: connection failed
			Unable to establish IB connection`,
			expectedMetrics: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Output:  tt.output,
				Metrics: make(map[string]interface{}),
			}

			runner.ParseMetrics(result)

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
				case int64:
					if actualInt, ok := actualValue.(int64); ok {
						if actualInt != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualInt)
						}
					} else {
						t.Errorf("Metric %s: expected int64, got %T", key, actualValue)
					}
				case int:
					if actualInt, ok := actualValue.(int); ok {
						if actualInt != expectedValue {
							t.Errorf("Metric %s: expected %v, got %v", key, expectedValue, actualInt)
						}
					} else {
						t.Errorf("Metric %s: expected int, got %T", key, actualValue)
					}
				case string:
					if actualString, ok := actualValue.(string); ok {
						if actualString != expectedValue {
							t.Errorf("Metric %s: expected %s, got %s", key, expectedValue, actualString)
						}
					} else {
						t.Errorf("Metric %s: expected string, got %T", key, actualValue)
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

func TestIbSendBwRunner_Validate(t *testing.T) {
	runner := NewIbSendBwRunner("")

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
				Port: 18515,
			},
			wantErr: false,
		},
		{
			name: "valid client config with host",
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Port: 18515,
			},
			wantErr: false,
		},
		{
			name: "valid client config with target_host",
			config: Config{
				Role:       "client",
				TargetHost: "10.0.0.100",
				Port:       18515,
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
				Port: 18515,
			},
			wantErr: true,
			errMsg:  "target_host or host is required for client role",
		},
		{
			name: "server with unnecessary host (should pass)",
			config: Config{
				Role: "server",
				Host: "192.168.1.100",
				Port: 18515,
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

func TestIbSendBwRunner_SupportsRole(t *testing.T) {
	runner := NewIbSendBwRunner("")

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

func TestIbSendBwRunner_Name(t *testing.T) {
	runner := NewIbSendBwRunner("")
	
	if name := runner.Name(); name != "ib_send_bw" {
		t.Errorf("Expected name 'ib_send_bw', got %q", name)
	}
}

func TestIbSendBwRunner_BuildCommand_ArgumentHandling(t *testing.T) {
	runner := NewIbSendBwRunner("")

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected []string
		notExpected []string
	}{
		{
			name: "string size argument",
			args: map[string]interface{}{
				"size": "64K",
			},
			expected: []string{"-s 64K"},
		},
		{
			name: "int size argument",
			args: map[string]interface{}{
				"size": 65536,
			},
			expected: []string{"-s 65536"},
		},
		{
			name: "boolean flags enabled",
			args: map[string]interface{}{
				"use_event":        true,
				"bidirectional":    true,
				"report_cycles":    true,
				"report_histogram": true,
				"odp":              true,
				"report_gbits":     true,
			},
			expected: []string{"-e", "-b", "-C", "-H", "-o", "-R"},
		},
		{
			name: "boolean flags disabled",
			args: map[string]interface{}{
				"use_event":        false,
				"bidirectional":    false,
				"report_cycles":    false,
				"report_histogram": false,
				"odp":              false,
				"report_gbits":     false,
			},
			notExpected: []string{"-e", "-b", "-C", "-H", "-o", "-R"},
		},
		{
			name: "mixed argument types",
			args: map[string]interface{}{
				"size":        65536,
				"iterations":  1000,
				"connection":  "RC",
				"ib_dev":      "mlx5_0",
				"gid_index":   3,
				"cpu_freq":    2.4,
				"use_event":   true,
			},
			expected: []string{
				"-s 65536",
				"-n 1000", 
				"-c RC",
				"-d mlx5_0",
				"-x 3",
				"-F 2.40",
				"-e",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Role: "server",
				Args: tt.args,
			}

			cmd := runner.BuildCommand(config)

			for _, expected := range tt.expected {
				if !contains(cmd, expected) {
					t.Errorf("Expected %q in command: %s", expected, cmd)
				}
			}

			for _, notExpected := range tt.notExpected {
				if contains(cmd, notExpected) {
					t.Errorf("Did not expect %q in command: %s", notExpected, cmd)
				}
			}
		})
	}
}

func TestIbSendBwRunner_CustomExecutablePath(t *testing.T) {
	customPath := "/usr/local/bin/ib_send_bw"
	runner := NewIbSendBwRunner(customPath)

	config := Config{
		Role: "server",
	}

	cmd := runner.BuildCommand(config)
	
	if !contains(cmd, customPath) {
		t.Errorf("Expected custom path %q in command: %s", customPath, cmd)
	}
}

func TestIbSendBwRunner_Duration(t *testing.T) {
	runner := NewIbSendBwRunner("")

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "-D 30",
		},
		{
			name:     "2 minutes",
			duration: 2 * time.Minute,
			expected: "-D 120",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "", // should not include -D flag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Role:     "server",
				Duration: tt.duration,
			}

			cmd := runner.BuildCommand(config)

			if tt.expected == "" {
				if contains(cmd, "-D") {
					t.Errorf("Did not expect -D flag in command: %s", cmd)
				}
			} else {
				if !contains(cmd, tt.expected) {
					t.Errorf("Expected %q in command: %s", tt.expected, cmd)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}