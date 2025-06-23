package runner

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEnvPrefix(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "no environment variables",
			config: Config{
				Env: nil,
			},
			expected: "",
		},
		{
			name: "empty environment map",
			config: Config{
				Env: map[string]string{},
			},
			expected: "",
		},
		{
			name: "single environment variable",
			config: Config{
				Env: map[string]string{
					"LD_LIBRARY_PATH": "/usr/local/lib",
				},
			},
			expected: "LD_LIBRARY_PATH=/usr/local/lib ",
		},
		{
			name: "multiple environment variables",
			config: Config{
				Env: map[string]string{
					"LD_LIBRARY_PATH": "/usr/local/lib",
					"RDMA_DEBUG":      "1",
				},
			},
			expected: "LD_LIBRARY_PATH=/usr/local/lib RDMA_DEBUG=1 ",
		},
		{
			name: "environment variable with spaces",
			config: Config{
				Env: map[string]string{
					"PATH": "/usr/local/bin:/usr/bin",
					"MSG":  "hello world",
				},
			},
			expected: "MSG='hello world' PATH=/usr/local/bin:/usr/bin ",
		},
		{
			name: "environment variable with special characters",
			config: Config{
				Env: map[string]string{
					"SPECIAL": "value with $pecial chars",
				},
			},
			expected: "SPECIAL='value with $pecial chars' ",
		},
		{
			name: "environment variable with quotes",
			config: Config{
				Env: map[string]string{
					"QUOTED": "value with 'quotes'",
				},
			},
			expected: "QUOTED='value with '\"'\"'quotes'\"'\"'' ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEnvPrefix(tt.config)
			if result != tt.expected {
				t.Errorf("buildEnvPrefix() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestRunnerEnvironmentVariableIntegration(t *testing.T) {
	tests := []struct {
		name     string
		runner   Runner
		config   Config
		expected string
	}{
		{
			name:   "ib_send_bw with LD_LIBRARY_PATH",
			runner: NewIbSendBwRunner(""),
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Env: map[string]string{
					"LD_LIBRARY_PATH": "/opt/mellanox/lib",
				},
			},
			expected: "LD_LIBRARY_PATH=/opt/mellanox/lib ib_send_bw 192.168.1.100",
		},
		{
			name:   "iperf3 with multiple environment variables",
			runner: NewIperf3Runner(""),
			config: Config{
				Role: "server",
				Port: 5201,
				Duration: 30 * time.Second,
				Env: map[string]string{
					"LD_LIBRARY_PATH": "/usr/local/lib",
					"NUMA_POLICY":     "preferred",
				},
			},
			expected: "LD_LIBRARY_PATH=/usr/local/lib NUMA_POLICY=preferred iperf3 -s -p 5201 -t 30 -J",
		},
		{
			name:   "ibperf with RDMA environment variables",
			runner: NewIbperfRunner(""),
			config: Config{
				Role: "client",
				Host: "192.168.1.100",
				Env: map[string]string{
					"LD_LIBRARY_PATH": "/opt/rdma/lib",
					"UCX_TLS":         "rc_x,ud_x",
				},
				Args: map[string]interface{}{
					"verifier": true,
				},
			},
			expected: "LD_LIBRARY_PATH=/opt/rdma/lib UCX_TLS=rc_x,ud_x ibperf 192.168.1.100 -V",
		},
		{
			name:   "no environment variables",
			runner: NewIbSendBwRunner(""),
			config: Config{
				Role: "server",
			},
			expected: "ib_send_bw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.runner.BuildCommand(tt.config)
			if result != tt.expected {
				t.Errorf("BuildCommand() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestEnvironmentVariableOrdering(t *testing.T) {
	// Test that environment variables are consistently ordered
	config := Config{
		Env: map[string]string{
			"Z_VAR":           "last",
			"A_VAR":           "first", 
			"LD_LIBRARY_PATH": "/usr/local/lib",
			"M_VAR":           "middle",
		},
	}

	result := buildEnvPrefix(config)
	
	// Variables should be alphabetically ordered
	expectedOrder := []string{"A_VAR=first", "LD_LIBRARY_PATH=/usr/local/lib", "M_VAR=middle", "Z_VAR=last"}
	
	for i, expected := range expectedOrder {
		if !strings.Contains(result, expected) {
			t.Errorf("Environment prefix missing expected variable: %s", expected)
		}
		
		// Check ordering by finding positions
		pos := strings.Index(result, expected)
		if i > 0 {
			prevPos := strings.Index(result, expectedOrder[i-1])
			if pos < prevPos {
				t.Errorf("Environment variables are not in alphabetical order: %s should come after %s", expected, expectedOrder[i-1])
			}
		}
	}
}