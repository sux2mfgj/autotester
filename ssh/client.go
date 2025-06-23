package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config represents SSH connection configuration
type Config struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	KeyPath         string        `yaml:"key_path"`
	Password        string        `yaml:"password,omitempty"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
	CommandTimeout  time.Duration `yaml:"command_timeout"`
}

// Client wraps SSH client functionality
type Client struct {
	config *Config
	client *ssh.Client
}

// Result represents the result of a remote command execution
type Result struct {
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// NewClient creates a new SSH client
func NewClient(config *Config) *Client {
	if config.Port == 0 {
		config.Port = 22
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 30 * time.Second
	}
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 300 * time.Second
	}
	
	return &Client{
		config: config,
	}
}

// Connect establishes an SSH connection
func (c *Client) Connect(ctx context.Context) error {
	if c.client != nil {
		return nil // Already connected
	}
	
	// Prepare authentication
	var authMethods []ssh.AuthMethod
	
	// Key-based authentication
	if c.config.KeyPath != "" {
		key, err := c.loadPrivateKey(c.config.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}
	
	// Password authentication
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}
	
	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method provided")
	}
	
	// SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, implement proper host key verification
		Timeout:         c.config.ConnectTimeout,
	}
	
	// Connect
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	
	// Use context for connection timeout
	conn, err := c.dialWithContext(ctx, "tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	
	c.client = conn
	return nil
}

// ExecuteCommand runs a command on the remote host
func (c *Client) ExecuteCommand(ctx context.Context, command string) (*Result, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	
	// Create session
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()
	
	result := &Result{}
	
	// Create context with timeout for command execution
	cmdCtx, cancel := context.WithTimeout(ctx, c.config.CommandTimeout)
	defer cancel()
	
	// Channel to receive command completion
	done := make(chan error, 1)
	
	go func() {
		// Capture output
		output, err := session.CombinedOutput(command)
		result.Output = string(output)
		
		if err != nil {
			result.Error = err.Error()
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			}
		}
		
		done <- err
	}()
	
	// Wait for command completion or context cancellation
	select {
	case err := <-done:
		return result, err
	case <-cmdCtx.Done():
		// Try to close the session to terminate the command
		session.Close()
		return nil, fmt.Errorf("command timed out: %w", cmdCtx.Err())
	}
}

// ExecuteCommandAsync runs a command without waiting for completion
func (c *Client) ExecuteCommandAsync(ctx context.Context, command string) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	
	// Start the command without waiting
	if err := session.Start(command); err != nil {
		session.Close()
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Close session in a goroutine to avoid blocking
	go func() {
		session.Wait()
		session.Close()
	}()
	
	return nil
}

// Config returns the SSH configuration
func (c *Client) Config() *Config {
	return c.config
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// loadPrivateKey loads a private key from file
func (c *Client) loadPrivateKey(keyPath string) (ssh.Signer, error) {
	// Expand home directory
	if keyPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = filepath.Join(home, keyPath[1:])
	}
	
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	
	// Try to parse the key
	key, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, err
	}
	
	return key, nil
}

// dialWithContext provides context-aware dialing
func (c *Client) dialWithContext(ctx context.Context, network, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	// Create a dialer with context
	dialer := &net.Dialer{
		Timeout: config.Timeout,
	}
	
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	
	// Create SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	
	return ssh.NewClient(sshConn, chans, reqs), nil
}