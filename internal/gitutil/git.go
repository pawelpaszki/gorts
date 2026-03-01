package gitutil

import (
	"fmt"
	"strings"

	"github.com/pawelpaszki/gorts/internal/exec"
)

// VerifyCleanRepo ensures that function checksums are not tainted by any
// uncommitted files, hence we check the status of the repo to make sure
// that the git state is clean before making any ast computations
func VerifyCleanRepo(repoPath string) error {
	// Check for uncommitted changes to Go files
	stdout, _, err := exec.Run(repoPath, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if stdout == "" {
		return nil // no changes - repo clean
	}

	// Check each changed file
	var taintedFiles []string
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		// ignore output shorter than 2 chars
		if len(line) < 3 {
			continue
		}
		// sample line from --porcelain looks like this:
		//  M internal/model/mapping.go
		file := strings.TrimSpace(line[2:])
		// no go files changes allowed when calculating function checksums
		if isTaintingFile(file) {
			taintedFiles = append(taintedFiles, file)
		}
	}

	if len(taintedFiles) > 0 {
		return fmt.Errorf("uncommitted changes to Go-related files would taint checksums:\n  - %s\nPlease commit or stash changes before generating mapping",
			strings.Join(taintedFiles, "\n  - "))
	}

	return nil
}

// VerifyAtCommit ensures repo HEAD matches expected commit
func VerifyAtCommit(repoPath, expectedCommit string) error {
	stdout, _, err := exec.Run(repoPath, "git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	currentCommit := strings.TrimSpace(stdout)
	if currentCommit != expectedCommit {
		return fmt.Errorf("repo is at commit %s but baseline was generated at %s\nCheckout baseline commit or regenerate baseline at current HEAD",
			currentCommit[:12], expectedCommit[:12])
	}

	return nil
}

func isTaintingFile(file string) bool {
	taintPatterns := []string{".go", "go.mod", "go.sum"}
	for _, pattern := range taintPatterns {
		if strings.HasSuffix(file, pattern) {
			return true
		}
	}
	return false
}
