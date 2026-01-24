package model

import (
	"fmt"
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

func ValidateTestManifest(m *TestManifest) error {
	if m.GeneratedAt.IsZero() {
		return fmt.Errorf("missing generated_at")
	}
	if m.CommitSHA == "" {
		return fmt.Errorf("missing commit_sha")
	}
	if len(m.TestSuites) == 0 {
		return fmt.Errorf("missing test_suites")
	}
	return nil
}
