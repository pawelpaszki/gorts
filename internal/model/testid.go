package model

import (
	"path/filepath"
	"strings"
)

// QualifyTestName returns "directory/TestName" to uniquely identify a test
func QualifyTestName(directory, testName string) string {
	// Normalize directory (remove leading ./ or trailing /)
	dir := strings.TrimPrefix(directory, "./")
	dir = strings.TrimSuffix(dir, "/")
	return dir + "/" + testName
}

// ParseQualifiedTest splits "directory/testName" back into parts
func ParseQualifiedTest(qualified string) (directory, testName string) {
	idx := strings.LastIndex(qualified, "/")
	if idx == -1 {
		return "", qualified // Fallback: no directory
	}
	return qualified[:idx], qualified[idx+1:]
}

// DirectoryForGoTest returns the directory in go test format
func DirectoryForGoTest(directory string) string {
	dir := strings.TrimPrefix(directory, "./")
	if !strings.HasPrefix(dir, ".") {
		dir = "./" + dir
	}
	return dir + "/..."
}

// MatchesRunAllPattern checks if a file matches any run-all pattern and thus all tests need to be executed
func MatchesRunAllPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		// check for wildcard matches first
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, file)
			if matched {
				return true
			}
			// try to match against just a filename
			matched, _ = filepath.Match(pattern, filepath.Base(file))
			if matched {
				return true
			}
		} else {
			// Exact match or suffix match
			if file == pattern || strings.HasSuffix(file, "/"+pattern) || file == pattern {
				return true
			}
		}
	}
	return false
}

// CheckRunAllTrigger checks if any changed files should trigger 'running all tests' and returns first match (if any)
func CheckRunAllTrigger(changedFiles, patterns []string) (bool, string) {
	for _, file := range changedFiles {
		if MatchesRunAllPattern(file, patterns) {
			return true, file
		}
	}
	return false, ""
}
