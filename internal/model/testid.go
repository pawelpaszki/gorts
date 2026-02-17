package model

import (
	"strings"
)

// QualifyTestName returns "directory/TestName" to uniquely identify a test
func QualifyTestName(directory, testName string) string {
	// Normalize directory (remove leading ./ or trailing /)
	dir := strings.TrimPrefix(directory, "./")
	dir = strings.TrimSuffix(dir, "/")
	return dir + "/" + testName
}

// ParseQualifiedTest splits "directory/TestName" back into parts
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
