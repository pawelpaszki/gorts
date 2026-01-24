package model

import "time"

type TestResult struct {
	Directory  string `json:"directory"`
	TestName   string `json:"test_name"`
	Status     string `json:"status"` // "pass", "fail"
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type BaselineManifest struct {
	GeneratedAt time.Time    `json:"generated_at"`
	CommitSHA   string       `json:"commit_sha"`
	TestResults []TestResult `json:"test_results"`
}
