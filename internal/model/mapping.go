package model

import (
	"fmt"
	"time"
)

// CoverageMapping holds the bidirectional mapping between files and tests
type CoverageMapping struct {
	GeneratedAt time.Time `json:"generated_at"`
	CommitSHA   string    `json:"commit_sha"`

	// file level mapping
	FileToTests map[string][]string `json:"file_to_tests"` // all tests covering given file
	TestToFiles map[string][]string `json:"test_to_files"` // all files covered by given test

	// function level mapping
	FunctionToTests   map[string][]string `json:"function_to_tests,omitempty"`
	TestToFunctions   map[string][]string `json:"test_to_functions,omitempty"`
	FunctionChecksums map[string]string   `json:"function_checksums,omitempty"`

	Stats MappingStats `json:"stats"`
}

type MappingStats struct {
	TotalTests          int     `json:"total_tests"`
	TotalFiles          int     `json:"total_files"`
	TotalFunctions      int     `json:"total_functions,omitempty"`
	AvgFilesPerTest     float64 `json:"avg_files_per_test"`
	AvgTestsPerFile     float64 `json:"avg_tests_per_file"`
	AvgFunctionsPerTest float64 `json:"avg_functions_per_test,omitempty"`
}

func ValidateCoverageMapping(m *CoverageMapping) error {
	if m.GeneratedAt.IsZero() {
		return fmt.Errorf("missing generated_at")
	}
	if m.CommitSHA == "" {
		return fmt.Errorf("missing commit_sha")
	}
	if len(m.FileToTests) == 0 {
		return fmt.Errorf("file_to_tests mapping is empty")
	}
	if m.Stats.TotalTests == 0 {
		return fmt.Errorf("invalid stats: total_tests is 0")
	}
	if m.Stats.TotalFiles == 0 {
		return fmt.Errorf("invalid stats: total_files is 0")
	}
	return nil
}
