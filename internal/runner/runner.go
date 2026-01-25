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
	PreHook  PreHook
	PostHook PostHook
	Env      []string // e.g., []string{"KEY=value", "FOO=bar"}
}

func New() *Runner {
	return &Runner{}
}

func (r *Runner) RunSingleTest(directory, testName string) (*model.TestResult, error) {
	// Pre-hook
	if r.PreHook != nil {
		if err := r.PreHook(directory, testName); err != nil {
			return nil, fmt.Errorf("pre-hook failed: %w", err)
		}
	}

	start := time.Now()
	cmd := exec.Command("go", "test", "-v", "-run", fmt.Sprintf("^%s$", testName), "./...")
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), r.Env...)

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &model.TestResult{
		TestName:   testName,
		DurationMs: duration.Milliseconds(),
	}

	if err != nil {
		result.Status = "fail"
		result.Error = string(output)
	} else {
		result.Status = "pass"
	}

	// Post-hook
	if r.PostHook != nil {
		if err := r.PostHook(directory, testName, result); err != nil {
			return result, fmt.Errorf("post-hook failed: %w", err)
		}
	}

	return result, nil
}
