package config

import (
	"testing"

	"perf-runner/ssh"
)

func TestValidator_ValidateConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  *TestConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &TestConfig{
				Name:   "Valid Config",
				Runner: "ib_send_bw",
				Hosts: map[string]*HostConfig{
					"client1": {
						SSH: &ssh.Config{
							Host:    "192.168.1.101",
							User:    "testuser",
							KeyPath: "~/.ssh/id_rsa",
						},
						Role: "client",
					},
					"server1": {
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
						Name:   "Test 1",
						Client: "client1",
						Server: "server1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &TestConfig{
				Runner: "ib_send_bw",
				Hosts: map[string]*HostConfig{
					"client1": {
						SSH: &ssh.Config{
							Host:    "192.168.1.101",
							User:    "testuser",
							KeyPath: "~/.ssh/id_rsa",
						},
						Role: "client",
					},
				},
				Tests: []TestScenario{
					{
						Name:   "Test 1",
						Client: "client1",
						Server: "server1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing runner",
			config: &TestConfig{
				Name: "Config Without Runner",
				Hosts: map[string]*HostConfig{
					"client1": {
						SSH: &ssh.Config{
							Host:    "192.168.1.101",
							User:    "testuser",
							KeyPath: "~/.ssh/id_rsa",
						},
						Role: "client",
					},
				},
				Tests: []TestScenario{
					{
						Name:   "Test 1",
						Client: "client1",
						Server: "server1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty config",
			config: &TestConfig{},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Error("NewValidator should not return nil")
	}
}