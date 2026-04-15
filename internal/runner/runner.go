package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pawelpaszki/gorts/internal/model"
)

const retryDelay = 5 * time.Second

type PreHook func(directory, testName string) error
type PostHook func(directory, testName string, result *model.TestResult) error

type Runner struct {
	PreHook     PreHook
	PostHook    PostHook
	Env         []string
	MaxRetries  int
	TestBinary  string
	CoverageDir string
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

	if r.TestBinary != "" {
		result.CoveragePath = r.buildCoveragePath(directory, testName)
	}

	lastOutput, attempt, durationMs := r.executeWithRetries(directory, testName, result)

	result.DurationMs = durationMs

	if result.Status == "" {
		result.Status = "fail"
		result.Retries = r.MaxRetries
		result.Flaky = false
		result.Error = truncateOutput(lastOutput, 2000)
	} else {
		result.Retries = attempt
		result.Flaky = attempt > 0
	}

	return result, nil
}

func (r *Runner) executeWithRetries(directory, testName string, result *model.TestResult) (string, int, int64) {
	var lastOutput string
	var lastAttemptDurationMs int64

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("    [Retry %d/%d] %s\n", attempt, r.MaxRetries, testName)
			time.Sleep(retryDelay)
		}

		r.runPreHook(directory, testName)

		cmd := r.buildCommand(directory, testName, result.CoveragePath)

		// Measure only test execution time, excluding hooks
		testStart := time.Now()
		output, err := cmd.CombinedOutput()
		lastAttemptDurationMs = time.Since(testStart).Milliseconds()

		lastOutput = string(output)

		r.runPostHook(directory, testName, result)

		if err == nil {
			result.Status = "pass"
			return lastOutput, attempt, lastAttemptDurationMs
		}

		fmt.Printf("    [Attempt %d failed]\n", attempt+1)
	}

	return lastOutput, r.MaxRetries, lastAttemptDurationMs
}

func (r *Runner) buildCommand(directory, testName, coveragePath string) *exec.Cmd {
	var cmd *exec.Cmd

	if r.TestBinary != "" {
		cmd = r.buildTestBinaryCommand(directory, testName, coveragePath)
	} else {
		cmd = r.buildGoTestCommand(directory, testName)
	}

	cmd.Env = append(os.Environ(), r.Env...)
	return cmd
}

func (r *Runner) buildTestBinaryCommand(directory, testName, coveragePath string) *exec.Cmd {
	cmd := exec.Command(r.TestBinary,
		"-test.v",
		"-test.timeout", "30m",
		"-test.run", fmt.Sprintf("^%s$", testName),
		"-test.gocoverdir", coveragePath,
	)
	cmd.Dir = directory
	return cmd
}

func (r *Runner) buildGoTestCommand(directory, testName string) *exec.Cmd {
	cmd := exec.Command("go", "test", "-v",
		"-timeout", "30m",
		"-p", "1",
		"-parallel", "1",
		"-count", "1",
		"-run", fmt.Sprintf("^%s$", testName),
		"./...")
	cmd.Dir = directory
	return cmd
}

func (r *Runner) runPreHook(directory, testName string) {
	if r.PreHook == nil {
		return
	}
	if err := r.PreHook(directory, testName); err != nil {
		fmt.Printf("    [Warn] Pre-hook failed: %v\n", err)
	}
}

func (r *Runner) runPostHook(directory, testName string, result *model.TestResult) {
	if r.PostHook == nil {
		return
	}
	if err := r.PostHook(directory, testName, result); err != nil {
		fmt.Printf("    [Warn] Post-hook failed: %v\n", err)
	}
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
