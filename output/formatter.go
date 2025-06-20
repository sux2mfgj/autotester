package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"tester/coordinator"
	"tester/runner"
)

// Formatter handles result output formatting
type Formatter struct {
	jsonOutput bool
}

// NewFormatter creates a new output formatter
func NewFormatter(jsonOutput bool) *Formatter {
	return &Formatter{
		jsonOutput: jsonOutput,
	}
}

// OutputResults outputs test results in the requested format
func (f *Formatter) OutputResults(results []*coordinator.TestResult, totalDuration time.Duration) error {
	if f.jsonOutput {
		return f.outputJSON(results, totalDuration)
	}
	return f.outputText(results, totalDuration)
}

// outputJSON outputs results in JSON format
func (f *Formatter) outputJSON(results []*coordinator.TestResult, totalDuration time.Duration) error {
	// Enhance results with detailed failure information for JSON output
	enhancedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		enhancedResult := map[string]interface{}{
			"scenario_name": result.ScenarioName,
			"success":       result.Success,
			"duration":      result.Duration,
			"start_time":    result.StartTime,
			"end_time":      result.EndTime,
		}
		
		if result.ClientCommand != "" {
			enhancedResult["client_command"] = result.ClientCommand
		}
		if result.ServerCommand != "" {
			enhancedResult["server_command"] = result.ServerCommand
		}
		
		if result.Error != "" {
			enhancedResult["error"] = result.Error
		}
		
		if result.ClientResult != nil {
			clientInfo := map[string]interface{}{
				"success":   result.ClientResult.Success,
				"duration":  result.ClientResult.Duration,
				"exit_code": result.ClientResult.ExitCode,
			}
			
			if result.ClientResult.Output != "" {
				clientInfo["output"] = result.ClientResult.Output
			}
			if result.ClientResult.Error != "" {
				clientInfo["error"] = result.ClientResult.Error
			}
			if len(result.ClientResult.Metrics) > 0 {
				clientInfo["metrics"] = result.ClientResult.Metrics
			}
			
			enhancedResult["client_result"] = clientInfo
		}
		
		if result.ServerResult != nil {
			serverInfo := map[string]interface{}{
				"success":   result.ServerResult.Success,
				"duration":  result.ServerResult.Duration,
				"exit_code": result.ServerResult.ExitCode,
			}
			
			if result.ServerResult.Output != "" {
				serverInfo["output"] = result.ServerResult.Output
			}
			if result.ServerResult.Error != "" {
				serverInfo["error"] = result.ServerResult.Error
			}
			if len(result.ServerResult.Metrics) > 0 {
				serverInfo["metrics"] = result.ServerResult.Metrics
			}
			
			enhancedResult["server_result"] = serverInfo
		}
		
		enhancedResults[i] = enhancedResult
	}
	
	output := map[string]interface{}{
		"total_duration": totalDuration,
		"total_tests":    len(results),
		"passed":         f.countPassed(results),
		"failed":         f.countFailed(results),
		"results":        enhancedResults,
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputText outputs results in human-readable text format
func (f *Formatter) outputText(results []*coordinator.TestResult, totalDuration time.Duration) error {
	fmt.Printf("\n=== Test Results ===\n")
	fmt.Printf("Total Duration: %v\n", totalDuration)
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", f.countPassed(results))
	fmt.Printf("Failed: %d\n", f.countFailed(results))
	fmt.Println()
	
	for i, result := range results {
		fmt.Printf("%d. %s\n", i+1, result.ScenarioName)
		fmt.Printf("   Status: %s\n", f.getStatusString(result.Success))
		fmt.Printf("   Duration: %v\n", result.Duration)
		
		if result.Error != "" {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		
		if result.ClientResult != nil {
			fmt.Printf("   Client: %s\n", f.getStatusString(result.ClientResult.Success))
			
			// Show client command
			if result.ClientCommand != "" {
				fmt.Printf("   Client Command: %s\n", result.ClientCommand)
			}
			
			// Always show client output if available
			if result.ClientResult.Output != "" {
				fmt.Printf("   Client Output:\n")
				lines := strings.Split(result.ClientResult.Output, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						fmt.Printf("     %s\n", line)
					}
				}
			}
			
			// Show metrics for successful runs
			if result.ClientResult.Success && len(result.ClientResult.Metrics) > 0 {
				fmt.Printf("   Client Metrics:\n")
				for k, v := range result.ClientResult.Metrics {
					fmt.Printf("     %s: %v\n", k, v)
				}
			}
			
			// Show detailed error info for failed runs
			if !result.ClientResult.Success {
				if result.ClientResult.Error != "" {
					fmt.Printf("   Client Error: %s\n", result.ClientResult.Error)
				}
				if result.ClientResult.ExitCode != 0 {
					fmt.Printf("   Client Exit Code: %d\n", result.ClientResult.ExitCode)
				}
			}
		}
		
		if result.ServerResult != nil {
			fmt.Printf("   Server: %s\n", f.getStatusString(result.ServerResult.Success))
			
			// Show server command
			if result.ServerCommand != "" {
				fmt.Printf("   Server Command: %s\n", result.ServerCommand)
			}
			
			// Always show server output if available
			if result.ServerResult.Output != "" {
				fmt.Printf("   Server Output:\n")
				lines := strings.Split(result.ServerResult.Output, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						fmt.Printf("     %s\n", line)
					}
				}
			}
			
			// Show detailed error info for failed runs
			if !result.ServerResult.Success {
				if result.ServerResult.Error != "" {
					fmt.Printf("   Server Error: %s\n", result.ServerResult.Error)
				}
				if result.ServerResult.ExitCode != 0 {
					fmt.Printf("   Server Exit Code: %d\n", result.ServerResult.ExitCode)
				}
			}
		}
		
		fmt.Println()
	}
	
	return nil
}

// getStatusString returns a colored status string
func (f *Formatter) getStatusString(success bool) string {
	if success {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

// countPassed counts the number of passed tests
func (f *Formatter) countPassed(results []*coordinator.TestResult) int {
	count := 0
	for _, result := range results {
		if result.Success {
			count++
		}
	}
	return count
}

// countFailed counts the number of failed tests
func (f *Formatter) countFailed(results []*coordinator.TestResult) int {
	count := 0
	for _, result := range results {
		if !result.Success {
			count++
		}
	}
	return count
}

// outputCommandDetails outputs detailed failure information for a command
func (f *Formatter) outputCommandDetails(role string, result *runner.Result) {
	if result.Error != "" {
		fmt.Printf("     %s Error: %s\n", role, result.Error)
	}
	
	if result.ExitCode != 0 {
		fmt.Printf("     %s Exit Code: %d\n", role, result.ExitCode)
	}
	
	if result.Output != "" {
		fmt.Printf("     %s Output:\n", role)
		lines := strings.Split(result.Output, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("       %s\n", line)
			}
		}
	}
}