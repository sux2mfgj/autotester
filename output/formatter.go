package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tester/coordinator"
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
	output := map[string]interface{}{
		"total_duration": totalDuration,
		"total_tests":    len(results),
		"passed":         f.countPassed(results),
		"failed":         f.countFailed(results),
		"results":        results,
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
			if result.ClientResult.Success && len(result.ClientResult.Metrics) > 0 {
				fmt.Printf("   Client Metrics:\n")
				for k, v := range result.ClientResult.Metrics {
					fmt.Printf("     %s: %v\n", k, v)
				}
			}
		}
		
		if result.ServerResult != nil {
			fmt.Printf("   Server: %s\n", f.getStatusString(result.ServerResult.Success))
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