package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"tester/runner"
	"tester/ssh"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "test_config.yaml")
	configContent := `
name: "Test Configuration"
description: "Test description"
runner: "ib_send_bw"
timeout: 5m

hosts:
  test_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      port: 18515

  test_client:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"

tests:
  - name: "Basic Test"
    description: "Basic test description"
    client: "test_client"
    server: "test_server"
    config:
      duration: 30s
      args:
        size: 65536
        iterations: 1000
`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading config
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test basic fields
	if config.Name != "Test Configuration" {
		t.Errorf("Expected name 'Test Configuration', got %q", config.Name)
	}

	if config.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got %q", config.Description)
	}

	if config.Runner != "ib_send_bw" {
		t.Errorf("Expected runner 'ib_send_bw', got %q", config.Runner)
	}

	if config.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", config.Timeout)
	}

	// Test hosts
	if len(config.Hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(config.Hosts))
	}

	serverHost, exists := config.Hosts["test_server"]
	if !exists {
		t.Error("test_server host should exist")
	} else {
		if serverHost.SSH.Host != "192.168.1.100" {
			t.Errorf("Expected server host '192.168.1.100', got %q", serverHost.SSH.Host)
		}
		if serverHost.Role != "server" {
			t.Errorf("Expected server role 'server', got %q", serverHost.Role)
		}
		if serverHost.Runner.Port != 18515 {
			t.Errorf("Expected server port 18515, got %d", serverHost.Runner.Port)
		}
	}

	clientHost, exists := config.Hosts["test_client"]
	if !exists {
		t.Error("test_client host should exist")
	} else {
		if clientHost.SSH.Host != "192.168.1.101" {
			t.Errorf("Expected client host '192.168.1.101', got %q", clientHost.SSH.Host)
		}
		if clientHost.Role != "client" {
			t.Errorf("Expected client role 'client', got %q", clientHost.Role)
		}
	}

	// Test tests
	if len(config.Tests) != 1 {
		t.Errorf("Expected 1 test, got %d", len(config.Tests))
	}

	test := config.Tests[0]
	if test.Name != "Basic Test" {
		t.Errorf("Expected test name 'Basic Test', got %q", test.Name)
	}

	if test.Client != "test_client" {
		t.Errorf("Expected test client 'test_client', got %q", test.Client)
	}

	if test.Server != "test_server" {
		t.Errorf("Expected test server 'test_server', got %q", test.Server)
	}

	if test.Config.Duration != 30*time.Second {
		t.Errorf("Expected test duration 30s, got %v", test.Config.Duration)
	}

	if size, exists := test.Config.Args["size"]; !exists || size != 65536 {
		t.Errorf("Expected test size 65536, got %v", size)
	}

	if iterations, exists := test.Config.Args["iterations"]; !exists || iterations != 1000 {
		t.Errorf("Expected test iterations 1000, got %v", iterations)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create temporary invalid YAML file
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidContent := `
invalid: yaml: content:
  - missing
    - brackets
`

	err = os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestGetClientHost(t *testing.T) {
	config := &TestConfig{
		Hosts: map[string]*HostConfig{
			"client1": {Role: "client"},
			"server1": {Role: "server"},
			"client2": {Role: "client"},
		},
	}

	tests := []struct {
		name     string
		test     *TestScenario
		expected string
	}{
		{
			name:     "existing client",
			test:     &TestScenario{Client: "client1"},
			expected: "client1",
		},
		{
			name:     "non-existent client",
			test:     &TestScenario{Client: "nonexistent"},
			expected: "",
		},
		{
			name:     "empty client",
			test:     &TestScenario{Client: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetClientHost(tt.test)
			if tt.expected == "" {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("Expected host config, got nil")
				} else if result != config.Hosts[tt.expected] {
					t.Error("Got wrong host config")
				}
			}
		})
	}
}

func TestGetServerHost(t *testing.T) {
	config := &TestConfig{
		Hosts: map[string]*HostConfig{
			"client1": {Role: "client"},
			"server1": {Role: "server"},
			"server2": {Role: "server"},
		},
	}

	tests := []struct {
		name     string
		test     *TestScenario
		expected string
	}{
		{
			name:     "existing server",
			test:     &TestScenario{Server: "server1"},
			expected: "server1",
		},
		{
			name:     "non-existent server",
			test:     &TestScenario{Server: "nonexistent"},
			expected: "",
		},
		{
			name:     "empty server",
			test:     &TestScenario{Server: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetServerHost(tt.test)
			if tt.expected == "" {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("Expected host config, got nil")
				} else if result != config.Hosts[tt.expected] {
					t.Error("Got wrong host config")
				}
			}
		})
	}
}

func TestMergeRunnerConfig(t *testing.T) {
	hostConfig := &runner.Config{
		Port:     18515,
		Duration: 60 * time.Second,
		Args: map[string]interface{}{
			"size":       65536,
			"iterations": 1000,
			"ib_dev":     "mlx5_0",
		},
		Env: map[string]string{
			"HOST_ENV": "host_value",
		},
	}

	testConfig := &runner.Config{
		Duration: 30 * time.Second, // Override duration
		Args: map[string]interface{}{
			"iterations": 500,    // Override iterations
			"connection": "RC",   // Add new arg
		},
		Env: map[string]string{
			"TEST_ENV": "test_value", // Add new env
		},
	}

	config := &TestConfig{}
	result := config.MergeRunnerConfig(hostConfig, testConfig)

	// Test that port is preserved from host config
	if result.Port != 18515 {
		t.Errorf("Expected port 18515, got %d", result.Port)
	}

	// Test that duration is overridden by test config
	if result.Duration != 30*time.Second {
		t.Errorf("Expected duration 30s, got %v", result.Duration)
	}

	// Test args merging
	if size, exists := result.Args["size"]; !exists || size != 65536 {
		t.Errorf("Expected size 65536 from host config, got %v", size)
	}

	if iterations, exists := result.Args["iterations"]; !exists || iterations != 500 {
		t.Errorf("Expected iterations 500 from test config (override), got %v", iterations)
	}

	if connection, exists := result.Args["connection"]; !exists || connection != "RC" {
		t.Errorf("Expected connection RC from test config, got %v", connection)
	}

	if ibDev, exists := result.Args["ib_dev"]; !exists || ibDev != "mlx5_0" {
		t.Errorf("Expected ib_dev mlx5_0 from host config, got %v", ibDev)
	}

	// Test env merging
	if hostEnv, exists := result.Env["HOST_ENV"]; !exists || hostEnv != "host_value" {
		t.Errorf("Expected HOST_ENV host_value from host config, got %v", hostEnv)
	}

	if testEnv, exists := result.Env["TEST_ENV"]; !exists || testEnv != "test_value" {
		t.Errorf("Expected TEST_ENV test_value from test config, got %v", testEnv)
	}
}

func TestMergeRunnerConfig_NilConfigs(t *testing.T) {
	config := &TestConfig{}

	tests := []struct {
		name       string
		hostConfig *runner.Config
		testConfig *runner.Config
	}{
		{
			name:       "both nil",
			hostConfig: nil,
			testConfig: nil,
		},
		{
			name:       "host nil",
			hostConfig: nil,
			testConfig: &runner.Config{Port: 18515},
		},
		{
			name:       "test nil",
			hostConfig: &runner.Config{Port: 18515},
			testConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.MergeRunnerConfig(tt.hostConfig, tt.testConfig)
			
			// Should always return a valid config
			if result == nil {
				t.Error("MergeRunnerConfig should never return nil")
			}

			// Maps should be initialized
			if result.Args == nil {
				t.Error("Args map should be initialized")
			}

			if result.Env == nil {
				t.Error("Env map should be initialized")
			}
		})
	}
}

func TestMergeRunnerConfig_EmptyMaps(t *testing.T) {
	hostConfig := &runner.Config{
		Args: make(map[string]interface{}),
		Env:  make(map[string]string),
	}

	testConfig := &runner.Config{
		Args: make(map[string]interface{}),
		Env:  make(map[string]string),
	}

	config := &TestConfig{}
	result := config.MergeRunnerConfig(hostConfig, testConfig)

	// Should handle empty maps gracefully
	if result.Args == nil {
		t.Error("Args should not be nil")
	}

	if result.Env == nil {
		t.Error("Env should not be nil")
	}

	if len(result.Args) != 0 {
		t.Errorf("Expected empty Args map, got %d items", len(result.Args))
	}

	if len(result.Env) != 0 {
		t.Errorf("Expected empty Env map, got %d items", len(result.Env))
	}
}

func TestSaveConfig(t *testing.T) {
	// Create test config
	config := &TestConfig{
		Name:        "Test Config",
		Description: "Test Description",
		Runner:      "ib_send_bw",
		Timeout:     5 * time.Minute,
		Hosts: map[string]*HostConfig{
			"test_client": {
				SSH: &ssh.Config{
					Host:    "192.168.1.101",
					User:    "testuser",
					KeyPath: "~/.ssh/id_rsa",
				},
				Role: "client",
			},
			"test_server": {
				SSH: &ssh.Config{
					Host:    "192.168.1.100",
					User:    "testuser",
					KeyPath: "~/.ssh/id_rsa",
				},
				Role: "server",
			},
		},
		Tests: []TestScenario{
			{
				Name:   "Test Scenario",
				Client: "test_client",
				Server: "test_server",
			},
		},
	}

	// Create temp file
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "save_test.yaml")

	// Test saving
	err = config.SaveConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Test loading saved config
	loadedConfig, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify content
	if loadedConfig.Name != config.Name {
		t.Errorf("Expected name %q, got %q", config.Name, loadedConfig.Name)
	}

	if loadedConfig.Description != config.Description {
		t.Errorf("Expected description %q, got %q", config.Description, loadedConfig.Description)
	}

	if loadedConfig.Runner != config.Runner {
		t.Errorf("Expected runner %q, got %q", config.Runner, loadedConfig.Runner)
	}
}

func TestSaveConfig_InvalidPath(t *testing.T) {
	config := &TestConfig{
		Name: "Test Config",
	}

	// Test saving to invalid path
	err := config.SaveConfig("/invalid/path/config.yaml")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}