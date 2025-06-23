package envinfo

import (
	"context"
	"fmt"
	"os/exec"

	"perf-runner/ssh"
)

// LocalExecutor executes commands on the local system
type LocalExecutor struct{}

// NewLocalExecutor creates a new local command executor
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// Execute runs a command locally
func (e *LocalExecutor) Execute(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("local command execution failed: %w", err)
	}
	return string(output), nil
}

// RemoteExecutor executes commands on a remote system via SSH
type RemoteExecutor struct {
	sshClient *ssh.Client
}

// NewRemoteExecutor creates a new remote command executor
func NewRemoteExecutor(sshClient *ssh.Client) *RemoteExecutor {
	return &RemoteExecutor{
		sshClient: sshClient,
	}
}

// Execute runs a command on the remote system
func (e *RemoteExecutor) Execute(ctx context.Context, command string) (string, error) {
	result, err := e.sshClient.ExecuteCommand(ctx, command)
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Error)
	}

	return result.Output, nil
}