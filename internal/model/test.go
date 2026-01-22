package model

import (
	"time"
)

// TestSuite represents tests in a single directory
type TestSuite struct {
	Directory string   `json:"directory"`
	Tests     []string `json:"tests"`
}

// TestManifest is the top-level structure
type TestManifest struct {
	GeneratedAt time.Time   `json:"generated_at"`
	CommitSHA   string      `json:"commit_sha"`
	TestSuites  []TestSuite `json:"test_suites"`
}
