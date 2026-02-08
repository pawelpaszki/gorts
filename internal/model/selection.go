package model

import (
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

type SelectionStats struct {
	TotalTests       int     `json:"total_tests"`
	SelectedTests    int     `json:"selected_tests"`
	ChangedFiles     int     `json:"changed_files"`
	ReductionPercent float64 `json:"reduction_percent"`
}
