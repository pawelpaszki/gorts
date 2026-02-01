package runner

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pawelpaszki/gorts/internal/model"
)

type PreHook func(directory, testName string) error
type PostHook func(directory, testName string, result *model.TestResult) error

type Runner struct {
	PreHook    PreHook
	PostHook   PostHook
	Env        []string // e.g., []string{"KEY=value", "FOO=bar"}
	MaxRetries int      // Max retry attempts per test (0 = no retries)
}

func New() *Runner {
	return &Runner{}
}

func (r *Runner) RunSingleTest(directory, testName string) (*model.TestResult, error) {
	result := &model.TestResult{
		Directory: directory,
		TestName:  testName,
		Retries:   0,
		Flaky:     false,
	}

	var lastOutput string
	totalStart := time.Now()

	// Retry loop
	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("    [Retry %d/%d] %s\n", attempt, r.MaxRetries, testName)
			time.Sleep(5 * time.Second) // Brief delay before retry
		}

		// Pre-hook (run before each attempt)
		if r.PreHook != nil {
			if err := r.PreHook(directory, testName); err != nil {
				fmt.Printf("    [Warn] Pre-hook failed: %v\n", err)
				// Continue anyway - test might still work
			}
		}

		// Run the test
		cmd := exec.Command("go", "test", "-v",
			"-timeout", "30m",
			"-p", "1",
			"-parallel", "1",
			"-count", "1",
			"-run", fmt.Sprintf("^%s$", testName),
			"./...")
		cmd.Dir = directory
		cmd.Env = append(os.Environ(), r.Env...)

		output, err := cmd.CombinedOutput()
		lastOutput = string(output)

		// Post-hook (run after each attempt)
		if r.PostHook != nil {
			if hookErr := r.PostHook(directory, testName, result); hookErr != nil {
				fmt.Printf("    [Warn] Post-hook failed: %v\n", hookErr)
			}
		}

		if err == nil {
			// Test passed
			result.Status = "pass"
			result.Retries = attempt
			result.Flaky = attempt > 0 // Flaky if needed any retry to pass
			result.DurationMs = time.Since(totalStart).Milliseconds()
			return result, nil
		}

		// Test failed, will retry if attempts remain
		fmt.Printf("    [Attempt %d failed]\n", attempt+1)
	}

	// All attempts exhausted - test failed
	result.Status = "fail"
	result.Retries = r.MaxRetries
	result.Flaky = false
	result.Error = truncateOutput(lastOutput, 2000) // Limit error size
	result.DurationMs = time.Since(totalStart).Milliseconds()

	return result, nil
}

// truncateOutput limits output length for storage
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... [truncated]"
}
