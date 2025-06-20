package coordinator

import (
	"strings"
	"testing"
	"time"

	"tester/runner"
)

func TestIbSendBwRunner_BuildCommand(t *testing.T) {
	// Create actual ib_send_bw runner instance
	ibRunner := runner.NewIbSendBwRunner("")

	tests := []struct {
		name     string
		config   runner.Config
		expected map[string]string // expected flags and their values
		notExpected []string       // flags that should not be present
	}{
		{
			name: "basic server config",
			config: runner.Config{
				Role: "server",
				Port: 18515,
				Args: map[string]interface{}{
					"size":       65536,
					"iterations": 1000,
				},
			},
			expected: map[string]string{
				"-p": "18515",
				"-s": "65536",
				"-n": "1000",
			},
			notExpected: []string{"192.168.1.100"}, // no host for server
		},
		{
			name: "basic client config",
			config: runner.Config{
				Role:       "client",
				Host:       "192.168.1.100",
				TargetHost: "10.0.0.100",
				Port:       18515,
				Args: map[string]interface{}{
					"size":       4096,
					"iterations": 500,
				},
			},
			expected: map[string]string{
				"10.0.0.100": "", // target host should be present
				"-p":         "18515",
				"-s":         "4096",
				"-n":         "500",
			},
		},
		{
			name: "client with fallback to host",
			config: runner.Config{
				Role: "client",
				Host: "192.168.1.100",
				Port: 18515,
			},
			expected: map[string]string{
				"192.168.1.100": "", // should use Host when TargetHost is empty
				"-p":            "18515",
			},
		},
		{
			name: "comprehensive config with all parameters",
			config: runner.Config{
				Role:     "client",
				Host:     "192.168.1.100",
				Port:     18515,
				Duration: 30 * time.Second,
				Args: map[string]interface{}{
					"size":             65536,
					"iterations":       1000,
					"tx_depth":         128,
					"rx_depth":         256,
					"mtu":              4096,
					"qp":               4,
					"connection":       "RC",
					"inline":           64,
					"ib_dev":           "mlx5_0",
					"gid_index":        3,
					"sl":               1,
					"cpu_freq":         2.4,
					"use_event":        true,
					"bidirectional":    true,
					"report_cycles":    true,
					"report_histogram": true,
					"odp":              true,
					"report_gbits":     true,
				},
			},
			expected: map[string]string{
				"192.168.1.100": "",
				"-p":            "18515",
				"-D":            "30",
				"-s":            "65536",
				"-n":            "1000",
				"-t":            "128",
				"-r":            "256",
				"-m":            "4096",
				"-q":            "4",
				"-c":            "RC",
				"-I":            "64",
				"-d":            "mlx5_0",
				"-x":            "3",
				"-S":            "1",
				"-F":            "2.4",
				"-e":            "",
				"-b":            "",
				"-C":            "",
				"-H":            "",
				"-o":            "",
				"-R":            "",
			},
		},
		{
			name: "boolean flags disabled",
			config: runner.Config{
				Role: "server",
				Args: map[string]interface{}{
					"use_event":        false,
					"bidirectional":    false,
					"report_cycles":    false,
					"report_histogram": false,
					"odp":              false,
					"report_gbits":     false,
				},
			},
			notExpected: []string{"-e", "-b", "-C", "-H", "-o", "-R"},
		},
		{
			name: "missing ib_dev parameter test",
			config: runner.Config{
				Role: "server",
				Args: map[string]interface{}{
					"size":      65536,
					"ib_dev":    "mlx5_0",
					"gid_index": 3,
				},
			},
			expected: map[string]string{
				"-s": "65536",
				"-d": "mlx5_0", // This is the critical test for the bug we just fixed
				"-x": "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ibRunner.BuildCommand(tt.config)
			
			// Check that all expected flags are present
			for flag, expectedValue := range tt.expected {
				if !strings.Contains(cmd, flag) {
					t.Errorf("Expected flag %s not found in command: %s", flag, cmd)
				}
				
				if expectedValue != "" {
					expectedPattern := flag + " " + expectedValue
					if !strings.Contains(cmd, expectedPattern) {
						t.Errorf("Expected pattern '%s' not found in command: %s", expectedPattern, cmd)
					}
				}
			}
			
			// Check that unwanted flags are not present
			for _, flag := range tt.notExpected {
				if strings.Contains(cmd, flag) {
					t.Errorf("Unexpected flag %s found in command: %s", flag, cmd)
				}
			}
			
			// Verify command starts with ib_send_bw
			if !strings.HasPrefix(cmd, "ib_send_bw") {
				t.Errorf("Command should start with 'ib_send_bw', got: %s", cmd)
			}
		})
	}
}

// TestIbSendBwRunner_ParameterCoverage ensures all documented parameters are handled
func TestIbSendBwRunner_ParameterCoverage(t *testing.T) {
	// This test ensures that all parameters documented in RUNNER_PARAMETERS.md
	// are actually implemented in the runner
	
	ibRunner := runner.NewIbSendBwRunner("")
	
	// Define all parameters that should be supported
	allParameters := map[string]interface{}{
		"size":             65536,
		"iterations":       1000,
		"tx_depth":         128,
		"rx_depth":         256,
		"mtu":              4096,
		"qp":               4,
		"connection":       "RC",
		"inline":           64,
		"ib_dev":           "mlx5_0",         // Critical parameter we just fixed
		"gid_index":        3,                // Critical parameter
		"sl":               1,
		"cpu_freq":         2.4,
		"use_event":        true,
		"bidirectional":    true,
		"report_cycles":    true,
		"report_histogram": true,
		"odp":              true,
		"report_gbits":     true,
	}
	
	config := runner.Config{
		Role: "client",
		Host: "192.168.1.100",
		Port: 18515,
		Args: allParameters,
	}
	
	cmd := ibRunner.BuildCommand(config)
	
	// Define expected flag mappings
	expectedFlags := map[string]string{
		"size":             "-s",
		"iterations":       "-n",
		"tx_depth":         "-t",
		"rx_depth":         "-r",
		"mtu":              "-m",
		"qp":               "-q",
		"connection":       "-c",
		"inline":           "-I",
		"ib_dev":           "-d", // This was the missing one!
		"gid_index":        "-x",
		"sl":               "-S",
		"cpu_freq":         "-F",
		"use_event":        "-e",
		"bidirectional":    "-b",
		"report_cycles":    "-C",
		"report_histogram": "-H",
		"odp":              "-o",
		"report_gbits":     "-R",
	}
	
	// Check each parameter is properly converted to its flag
	for param, flag := range expectedFlags {
		if !strings.Contains(cmd, flag) {
			t.Errorf("Parameter '%s' should generate flag '%s' but flag not found in command: %s", param, flag, cmd)
		}
	}
	
	t.Logf("Generated command: %s", cmd)
}

// TestRunner_Registry tests the runner registration system
func TestRunner_Registry(t *testing.T) {
	// Test that ib_send_bw is automatically registered
	availableRunners := runner.GetRegistered()
	
	found := false
	for _, name := range availableRunners {
		if name == "ib_send_bw" {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("ib_send_bw runner should be auto-registered, available runners: %v", availableRunners)
	}
	
	// Test creating runner from registry
	runnerInstance, err := runner.Create("ib_send_bw")
	if err != nil {
		t.Fatalf("Failed to create ib_send_bw runner: %v", err)
	}
	
	if runnerInstance.Name() != "ib_send_bw" {
		t.Errorf("Expected runner name 'ib_send_bw', got: %s", runnerInstance.Name())
	}
	
	// Test unknown runner
	_, err = runner.Create("unknown_runner")
	if err == nil {
		t.Error("Expected error for unknown runner")
	}
}

// TestIbSendBwRunner_EdgeCases tests edge cases and error conditions
func TestIbSendBwRunner_EdgeCases(t *testing.T) {
	ibRunner := runner.NewIbSendBwRunner("")
	
	tests := []struct {
		name   string
		config runner.Config
		check  func(t *testing.T, cmd string)
	}{
		{
			name: "zero port",
			config: runner.Config{
				Role: "server",
				Port: 0,
			},
			check: func(t *testing.T, cmd string) {
				if strings.Contains(cmd, "-p") {
					t.Error("Port flag should not be present when port is 0")
				}
			},
		},
		{
			name: "zero duration",
			config: runner.Config{
				Role:     "server",
				Duration: 0,
			},
			check: func(t *testing.T, cmd string) {
				if strings.Contains(cmd, "-D") {
					t.Error("Duration flag should not be present when duration is 0")
				}
			},
		},
		{
			name: "empty string values",
			config: runner.Config{
				Role: "server",
				Args: map[string]interface{}{
					"ib_dev":     "",
					"connection": "",
				},
			},
			check: func(t *testing.T, cmd string) {
				// Empty string values should still generate flags
				if !strings.Contains(cmd, "-d") {
					t.Error("ib_dev flag should be present even with empty value")
				}
				if !strings.Contains(cmd, "-c") {
					t.Error("connection flag should be present even with empty value")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ibRunner.BuildCommand(tt.config)
			tt.check(t, cmd)
		})
	}
}

// TestIbSendBwRunner_RegressionIbDevBug tests the specific bug we just fixed
func TestIbSendBwRunner_RegressionIbDevBug(t *testing.T) {
	// This test specifically checks for the bug where ib_dev parameter
	// was missing from the command builder, causing it to not appear in
	// generated commands even though it was configured.
	
	ibRunner := runner.NewIbSendBwRunner("")
	
	config := runner.Config{
		Role: "server",
		Args: map[string]interface{}{
			"ib_dev":    "mlx5_0",
			"gid_index": 3,
		},
	}
	
	cmd := ibRunner.BuildCommand(config)
	
	// The bug was that ib_dev parameter was completely missing from command output
	if !strings.Contains(cmd, "-d") {
		t.Fatal("REGRESSION: ib_dev parameter (-d flag) is missing from command. This was the original bug!")
	}
	
	if !strings.Contains(cmd, "mlx5_0") {
		t.Fatal("REGRESSION: ib_dev value 'mlx5_0' is missing from command")
	}
	
	if !strings.Contains(cmd, "-x") {
		t.Fatal("gid_index parameter (-x flag) is missing from command")
	}
	
	if !strings.Contains(cmd, "3") {
		t.Fatal("gid_index value '3' is missing from command")
	}
	
	expectedPattern := "-d mlx5_0"
	if !strings.Contains(cmd, expectedPattern) {
		t.Fatalf("Expected pattern '%s' not found in command: %s", expectedPattern, cmd)
	}
	
	expectedPattern = "-x 3"
	if !strings.Contains(cmd, expectedPattern) {
		t.Fatalf("Expected pattern '%s' not found in command: %s", expectedPattern, cmd)
	}
	
	t.Logf("SUCCESS: ib_dev parameter correctly generates command: %s", cmd)
}

