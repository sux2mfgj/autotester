package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"perf-runner/config"
	"perf-runner/coordinator"
	"perf-runner/output"
	"perf-runner/runner"
)

const appVersion = "1.0.0"

// App represents the main application
type App struct {
	flags  *Flags
	logger *log.Logger
}

// NewApp creates a new application instance
func NewApp() *App {
	flags := NewFlags()
	
	// Setup logging
	logger := log.New(os.Stderr, "[perf-runner] ", log.LstdFlags)
	if !*flags.Verbose {
		logger.SetOutput(os.Stderr)
	}
	
	return &App{
		flags:  flags,
		logger: logger,
	}
}

// Run executes the main application logic
func (a *App) Run() error {
	if *a.flags.Version {
		fmt.Printf("perf-runner version %s\n", appVersion)
		return nil
	}
	
	// Load configuration
	a.logger.Printf("Loading configuration from %s", *a.flags.ConfigFile)
	cfg, err := config.LoadConfig(*a.flags.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Override timeout if specified
	if *a.flags.Timeout != 10*time.Minute {
		cfg.Timeout = *a.flags.Timeout
	}
	
	a.logger.Printf("Loaded configuration: %s", cfg.Name)
	if cfg.Description != "" {
		a.logger.Printf("Description: %s", cfg.Description)
	}
	
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle graceful shutdown
	a.setupSignalHandling(cancel)
	
	// Create coordinator
	coord := coordinator.NewCoordinator(cfg, a.logger)
	defer coord.Cleanup()
	
	// Register runners
	if err := a.registerRunners(coord, cfg); err != nil {
		return fmt.Errorf("failed to register runners: %w", err)
	}
	
	// Connect to hosts
	a.logger.Printf("Connecting to %d hosts...", len(cfg.Hosts))
	if err := coord.ConnectHosts(ctx); err != nil {
		return fmt.Errorf("failed to connect to hosts: %w", err)
	}
	
	// Run tests
	a.logger.Printf("Starting test execution...")
	startTime := time.Now()
	
	results, err := coord.RunAllTests(ctx)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}
	
	duration := time.Since(startTime)
	a.logger.Printf("Test execution completed in %v", duration)
	
	// Output results
	formatter := output.NewFormatter(*a.flags.JSONOutput)
	if err := formatter.OutputResults(results, duration); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}
	
	// Exit with appropriate code
	exitCode := a.calculateExitCode(results)
	if exitCode != 0 {
		a.logger.Printf("Some tests failed, exiting with code %d", exitCode)
		os.Exit(exitCode)
	}
	
	return nil
}

// setupSignalHandling configures graceful shutdown
func (a *App) setupSignalHandling(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		a.logger.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()
}

// registerRunners registers available runner implementations using auto-discovery
func (a *App) registerRunners(coord *coordinator.Coordinator, cfg *config.TestConfig) error {
	// Get custom binary path if configured
	binaryPath := cfg.GetBinaryPath(cfg.Runner)
	
	// Create runner instance from registry with custom path
	runnerInstance, err := runner.CreateWithPath(cfg.Runner, binaryPath)
	if err != nil {
		availableRunners := runner.GetRegistered()
		return fmt.Errorf("unsupported runner '%s'. Available runners: %v", cfg.Runner, availableRunners)
	}
	
	if binaryPath != "" {
		a.logger.Printf("Using custom binary path for %s: %s", cfg.Runner, binaryPath)
	}
	
	// Register with coordinator
	coord.RegisterRunner(cfg.Runner, runnerInstance)
	
	return nil
}

// calculateExitCode determines the appropriate exit code
func (a *App) calculateExitCode(results []*coordinator.TestResult) int {
	for _, result := range results {
		if !result.Success {
			return 1
		}
	}
	return 0
}