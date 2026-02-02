package model

import "time"

// CoverageMapping holds the bidirectional mapping between files and tests
type CoverageMapping struct {
	GeneratedAt time.Time           `json:"generated_at"`
	CommitSHA   string              `json:"commit_sha"`
	FileToTests map[string][]string `json:"file_to_tests"` // file.go → [Test1, Test2]
	TestToFiles map[string][]string `json:"test_to_files"` // Test1 → [file1.go, file2.go]
	Stats       MappingStats        `json:"stats"`
}

type MappingStats struct {
	TotalTests      int     `json:"total_tests"`
	TotalFiles      int     `json:"total_files"`
	AvgFilesPerTest float64 `json:"avg_files_per_test"`
	AvgTestsPerFile float64 `json:"avg_tests_per_file"`
}
