package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pawelpaszki/gorts/internal/model"
)

type PreHook func(directory, testName string) error
type PostHook func(directory, testName string, result *model.TestResult) error

type Runner struct {
	PreHook     PreHook
	PostHook    PostHook
	Env         []string
	MaxRetries  int
	TestBinary  string // Path to pre-built test binary
	CoverageDir string // Base directory for coverage output (used with TestBinary)
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
			time.Sleep(5 * time.Second)
		}

		// Pre-hook (run before each attempt)
		if r.PreHook != nil {
			if err := r.PreHook(directory, testName); err != nil {
				fmt.Printf("    [Warn] Pre-hook failed: %v\n", err)
			}
		}

		// Build the command based on mode
		var cmd *exec.Cmd

		if r.TestBinary != "" {
			// Instrumented binary mode - coverage collected automatically
			covPath := r.buildCoveragePath(directory, testName)
			result.CoveragePath = covPath

			cmd = exec.Command(r.TestBinary,
				"-test.v",
				"-test.timeout", "30m",
				"-test.run", fmt.Sprintf("^%s$", testName),
				"-test.gocoverdir", covPath,
			)
			cmd.Dir = directory
		} else {
			// Standard mode - no coverage (hooks SHOULD handle it)
			cmd = exec.Command("go", "test", "-v",
				"-timeout", "30m",
				"-p", "1",
				"-parallel", "1",
				"-count", "1",
				"-run", fmt.Sprintf("^%s$", testName),
				"./...")
			cmd.Dir = directory
		}

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
			result.Status = "pass"
			result.Retries = attempt
			result.Flaky = attempt > 0
			result.DurationMs = time.Since(totalStart).Milliseconds()
			return result, nil
		}

		fmt.Printf("    [Attempt %d failed]\n", attempt+1)
	}

	// All attempts exhausted
	result.Status = "fail"
	result.Retries = r.MaxRetries
	result.Flaky = false
	result.Error = truncateOutput(lastOutput, 2000)
	result.DurationMs = time.Since(totalStart).Milliseconds()

	return result, nil
}

// buildCoveragePath creates and returns the coverage directory for a test
func (r *Runner) buildCoveragePath(directory, testName string) string {
	if r.CoverageDir == "" {
		return ""
	}

	// Sanitize directory name for path
	sanitized := filepath.Base(directory)
	parent := filepath.Base(filepath.Dir(directory))
	if parent != "." && parent != "/" {
		sanitized = parent + "_" + sanitized
	}

	covPath := filepath.Join(r.CoverageDir, sanitized, testName)
	covPath, _ = filepath.Abs(covPath)
	os.MkdirAll(covPath, 0755)

	return covPath
}

// truncateOutput limits output length for storage
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... [truncated]"
}
