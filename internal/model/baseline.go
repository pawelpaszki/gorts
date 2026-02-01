package model

import (
	"fmt"
	"time"
)

type Summary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Flaky      int   `json:"flaky"` // Tests that passed after retry
	DurationMs int64 `json:"duration_ms"`
}

type TestResult struct {
	Directory    string `json:"directory"`
	TestName     string `json:"test_name"`
	Status       string `json:"status"` // "pass", "fail"
	DurationMs   int64  `json:"duration_ms"`
	Retries      int    `json:"retries"` // Number of retries needed
	Flaky        bool   `json:"flaky"`   // True if passed only after retry
	Error        string `json:"error,omitempty"`
	CoveragePath string `json:"coverage_path,omitempty"`
}

type TestSuiteResult struct {
	Directory   string       `json:"directory"`
	TestResults []TestResult `json:"test_results"`
	Summary     Summary      `json:"summary"`
}

type BaselineManifest struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	CommitSHA        string            `json:"commit_sha"`
	TestSuiteResults []TestSuiteResult `json:"test_suite_results"`
	Summary          Summary           `json:"summary"`
}

func ValidateBaselineManifest(m *BaselineManifest) error {
	if m.GeneratedAt.IsZero() {
		return fmt.Errorf("missing generated_at")
	}
	if m.CommitSHA == "" {
		return fmt.Errorf("missing commit_sha")
	}
	if len(m.TestSuiteResults) == 0 {
		return fmt.Errorf("missing test_suite_results")
	}
	if m.Summary.Total == 0 {
		return fmt.Errorf("summary is empty: no tests were run")
	}
	return nil
}
