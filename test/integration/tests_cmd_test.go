//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestTestsCmd_SingleDirectory(t *testing.T) {
	// output goes to /tmp/gorts-test/.cov/ (same folder for all test artifacts)
	outputFile := filepath.Join(outputDir(t), "manifest.json")

	// run: gorts tests --directories ./test/e2e --output <outputFile>
	// it needs to be executed from the gorts-demo directory
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e",
		"--output", outputFile,
	)

	// should not fail
	if exitCode != 0 {
		t.Fatalf("gorts tests failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// manifest should exist
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected output file %s to exist", outputFile)
	}

	// manifest should be readable
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var manifest struct {
		GeneratedAt string `json:"generated_at"`
		CommitSHA   string `json:"commit_sha"`
		TestSuites  []struct {
			Directory string   `json:"directory"`
			Tests     []string `json:"tests"`
		} `json:"test_suites"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// verify manifest fields below
	if manifest.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}

	if manifest.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}

	if len(manifest.TestSuites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(manifest.TestSuites))
	}

	suite := manifest.TestSuites[0]
	if suite.Directory != "./test/e2e" {
		t.Errorf("expected directory './test/e2e', got '%s'", suite.Directory)
	}

	if len(suite.Tests) == 0 {
		t.Error("expected at least one test in the suite")
	}
}

func TestTestsCmd_MultipleDirectories(t *testing.T) {
	// output goes to /tmp/gorts-test/.cov/ (same folder for all test artifacts)
	outputFile := filepath.Join(outputDir(t), "manifest.json")

	// run: gorts tests --directories ./test/e2e,./test/integration --output <outputFile>
	// it needs to be executed from the gorts-demo directory
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e,./test/integration",
		"--output", outputFile,
	)

	// should not fail
	if exitCode != 0 {
		t.Fatalf("gorts tests failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// manifest should exist
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected output file %s to exist", outputFile)
	}

	// manifest should be readable
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var manifest struct {
		GeneratedAt string `json:"generated_at"`
		CommitSHA   string `json:"commit_sha"`
		TestSuites  []struct {
			Directory string   `json:"directory"`
			Tests     []string `json:"tests"`
		} `json:"test_suites"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// verify manifest fields below
	if manifest.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}

	if manifest.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}

	if len(manifest.TestSuites) != 2 {
		t.Fatalf("expected 2 test suite, got %d", len(manifest.TestSuites))
	}

	// Build a set of found directories and match with expectations (2 dirs)
	found := make(map[string]bool)
	for _, suite := range manifest.TestSuites {
		found[suite.Directory] = true
		if len(suite.Tests) == 0 {
			t.Errorf("expected at least one test in %s", suite.Directory)
		}
	}
	// Check both expected directories exist
	if !found["./test/e2e"] {
		t.Error("expected ./test/e2e in test suites")
	}
	if !found["./test/integration"] {
		t.Error("expected ./test/integration in test suites")
	}
}

func TestTestsCmd_EmptyTestsDirectory(t *testing.T) {
	// output goes to /tmp/gorts-test/.cov/ (same folder for all test artifacts)
	outputFile := filepath.Join(outputDir(t), "manifest.json")

	// create empty directory (with no tests)
	emptyDirPath := filepath.Join(tempDir, "test/empty")
	if err := os.MkdirAll(emptyDirPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create empty dir: %v\n", err)
		os.Exit(1)
	}

	// run: gorts tests --directories ./test/empty --output <outputFile>
	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", emptyDirPath,
		"--output", outputFile,
	)

	// should fail for a directory with no tests
	if exitCode != 1 {
		t.Fatalf("gorts tests should fail with exit code for empty test dir %d",
			exitCode)
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestTestsCmd_InvalidDirectory(t *testing.T) {
	// output goes to /tmp/gorts-test/.cov/ (same folder for all test artifacts)
	outputFile := filepath.Join(outputDir(t), "manifest.json")

	// run: gorts tests --directories nonexistent --output <outputFile>
	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "nonexistent",
		"--output", outputFile,
	)

	// should fail for nonexistent directory
	if exitCode != 1 {
		t.Fatalf("gorts tests should fail with exit code for empty test dir %d",
			exitCode)
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestTestsCmd_MissingMandatoryFlag_Output(t *testing.T) {
	// run: gorts tests --directories ./test/e2e
	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e",
	)

	// should fail
	if exitCode == 0 {
		t.Fatalf("gorts tests should fail with missing mandatory output flag")
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestTestsCmd_MissingMandatoryFlag_Directories(t *testing.T) {
	outputFile := filepath.Join(outputDir(t), "manifest.json")

	// run: gorts tests --output <outputFile>
	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--output", outputFile,
	)

	// should fail
	if exitCode == 0 {
		t.Fatalf("gorts tests should fail with missing mandatory output flag")
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}
