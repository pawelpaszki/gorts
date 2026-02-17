package model

import (
	"fmt"
	"time"
)

type Selection struct {
	GeneratedAt   time.Time      `json:"generated_at"`
	FromCommit    string         `json:"from_commit"`
	ToCommit      string         `json:"to_commit"`
	ChangedFiles  []string       `json:"changed_files"`
	SelectedTests []SelectedTest `json:"selected_tests"`
	Stats         SelectionStats `json:"stats"`
}

type SelectedTest struct {
	Directory string `json:"directory"`
	TestName  string `json:"test_name"`
}

// Qualified returns the unique identifier "directory/TestName"
func (s SelectedTest) Qualified() string {
	return QualifyTestName(s.Directory, s.TestName)
}

// ForGoTestRun returns the -run flag pattern
func (s SelectedTest) ForGoTestRun() string {
	return fmt.Sprintf("^%s$", s.TestName)
}

type SelectionStats struct {
	TotalTests       int     `json:"total_tests"`
	SelectedTests    int     `json:"selected_tests"`
	ChangedFiles     int     `json:"changed_files"`
	ReductionPercent float64 `json:"reduction_percent"`
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
