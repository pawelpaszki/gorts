package model

import (
	"fmt"
	"time"

	"github.com/pawelpaszki/gorts/internal/helpers"
)

// OutOfScopeTestFiles contains ignored tests (not in the baseline - likely unit tests)
type Selection struct {
	GeneratedAt         time.Time      `json:"generated_at"`
	FromCommit          string         `json:"from_commit"`
	ToCommit            string         `json:"to_commit"`
	ChangedFiles        []string       `json:"changed_files"`
	SelectedTests       []SelectedTest `json:"selected_tests"`
	OutOfScopeTestFiles []string       `json:"out_of_scope_test_files,omitempty"`
	Stats               SelectionStats `json:"stats"`
}

type SelectedTest struct {
	Directory string `json:"directory"`
	TestName  string `json:"test_name"`
}

// Qualified returns the unique identifier "directory/TestName"
func (s SelectedTest) Qualified() string {
	return helpers.QualifyTestName(s.Directory, s.TestName)
}

// ForGoTestRun returns the -run flag pattern
func (s SelectedTest) ForGoTestRun() string {
	return fmt.Sprintf("^%s$", s.TestName)
}

// since coverage does not detect new tests
// this needs to be handled separately
// each git diff between the baseline commit and current commit
// might also include new tests, hence ChangedTestFiles and NewTests are needed
// additionaly - we only care about directories from the baseline - there might be some
// tests (unit and/ or e2e) that we ignore (no baseline entry)
type SelectionStats struct {
	TotalTests          int     `json:"total_tests"`
	SelectedTests       int     `json:"selected_tests"`
	ChangedFiles        int     `json:"changed_files"`
	ChangedTestFiles    int     `json:"changed_test_files"`
	OutOfScopeTestFiles int     `json:"out_of_scope_test_files"`
	NewTests            int     `json:"new_tests"`
	ReductionPercent    float64 `json:"reduction_percent"`
}

func ValidateSelection(s *Selection) error {
	if s.GeneratedAt.IsZero() {
		return fmt.Errorf("missing generated_at")
	}
	if s.FromCommit == "" {
		return fmt.Errorf("missing from_commit")
	}
	if s.ToCommit == "" {
		return fmt.Errorf("missing to_commit")
	}
	return nil
}
