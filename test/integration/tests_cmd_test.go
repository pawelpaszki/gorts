//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestsCmd_SingleDirectory(t *testing.T) {
	t.Log("running gorts tests with single directory: ./test/e2e")

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

	t.Logf("manifest created at %s", outputFile)

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

	t.Log("verifying manifest structure")

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

	t.Logf("found %d tests in %s", len(suite.Tests), suite.Directory)
}

func TestTestsCmd_MultipleDirectories(t *testing.T) {
	t.Log("running gorts tests with multiple directories: ./test/e2e, ./test/integration")

	outputFile := filepath.Join(outputDir(t), "manifest.json")

	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e,./test/integration",
		"--output", outputFile,
	)

	if exitCode != 0 {
		t.Fatalf("gorts tests failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("manifest created at %s", outputFile)

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected output file %s to exist", outputFile)
	}

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

	t.Log("verifying manifest structure")

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
		t.Logf("found %d tests in %s", len(suite.Tests), suite.Directory)
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
	t.Log("running gorts tests with empty directory (expecting failure)")

	outputFile := filepath.Join(outputDir(t), "manifest.json")

	emptyDirPath := filepath.Join(tempDir, "test/empty")
	if err := os.MkdirAll(emptyDirPath, 0755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", emptyDirPath,
		"--output", outputFile,
	)

	if exitCode != 1 {
		t.Fatalf("gorts tests should fail with exit code for empty test dir %d",
			exitCode)
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	if !strings.Contains(stderr, "failed to list tests") {
		t.Errorf("expected 'failed to list tests' in stderr, got: %s", stderr)
	}

	t.Log("got expected error for empty directory")
}

func TestTestsCmd_InvalidDirectory(t *testing.T) {
	t.Log("running gorts tests with nonexistent directory (expecting failure)")

	outputFile := filepath.Join(outputDir(t), "manifest.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "nonexistent",
		"--output", outputFile,
	)

	if exitCode != 1 {
		t.Fatalf("gorts tests should fail with exit code for empty test dir %d",
			exitCode)
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	if !strings.Contains(stderr, "failed to list tests") {
		t.Errorf("expected 'failed to list tests' in stderr, got: %s", stderr)
	}

	t.Log("got expected error for invalid directory")
}

func TestTestsCmd_MissingMandatoryFlag_Output(t *testing.T) {
	t.Log("running gorts tests without --output flag (expecting failure)")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e",
	)

	if exitCode == 0 {
		t.Fatalf("gorts tests should fail with missing mandatory output flag")
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	if !strings.Contains(stderr, "required flag") || !strings.Contains(stderr, "output") {
		t.Errorf("expected error about required 'output' flag, got: %s", stderr)
	}

	t.Log("got expected error for missing --output flag")
}

func TestTestsCmd_MissingMandatoryFlag_Directories(t *testing.T) {
	t.Log("running gorts tests without --directories flag (expecting failure)")

	outputFile := filepath.Join(outputDir(t), "manifest.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--output", outputFile,
	)

	if exitCode == 0 {
		t.Fatalf("gorts tests should fail with missing mandatory output flag")
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	if !strings.Contains(stderr, "required flag") || !strings.Contains(stderr, "directories") {
		t.Errorf("expected error about required 'directories' flag, got: %s", stderr)
	}

	t.Log("got expected error for missing --directories flag")
}
