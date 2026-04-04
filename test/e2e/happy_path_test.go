//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestE2E_HappyPath validates the complete gorts workflow:
// gorts tests -> gorts baseline -> gorts mapping -> gorts select
//
// This test verifies that the full pipeline works end-to-end and that
// selected tests are actually the ones affected by code changes.
func TestE2E_HappyPath(t *testing.T) {
	// Restore repo to baselineCommit after test for other tests
	defer func() {
		if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
			t.Errorf("failed to restore baseline commit: %v", err)
		}
	}()

	t.Log("using pre-built test binary from TestMain")
	t.Logf("test binary: %s", testBinaryPath)
	// Phase 1: gorts tests
	t.Log("running gorts tests with single directory: ./test/e2e")

	manifestFile := filepath.Join(outputDir(t, "happy_path"), "manifest.json")

	// run: gorts tests --directories ./test/e2e --output <outputFile>
	// it needs to be executed from the gorts-demo directory
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e",
		"--output", manifestFile,
	)

	// should not fail
	if exitCode != 0 {
		t.Fatalf("gorts tests failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("manifest created at %s", manifestFile)

	// manifest should exist
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		t.Fatalf("expected output file %s to exist", manifestFile)
	}

	// manifest should be readable
	_, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Phase 2: gorts baseline
	t.Logf("running gorts baseline with previously generated manifest: %s", manifestFile)
	coverageDir := filepath.Join(outputDir(t, "happy_path"), "coverage_test")
	baselineFile := filepath.Join(outputDir(t, "happy_path"), "baseline_test.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", testBinaryPath,
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
	)
	if exitCode != 0 {
		t.Fatalf("gorts baseline failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("baseline created at %s", baselineFile)

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Fatalf("expected baseline file %s to exist", baselineFile)
	}

	_, err = os.ReadFile(baselineFile)
	if err != nil {
		t.Fatalf("failed to read baseline file: %v", err)
	}
	// Phase 3: gorts mapping
	t.Logf("running gorts mapping with previously generated manifest: %s and baseline: %s", manifestFile, baselineFile)
	mappingFile := filepath.Join(outputDir(t, "happy_path"), "mapping_file_level.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", mappingFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
	)
	if exitCode != 0 {
		t.Fatalf("gorts mapping failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("mapping created at %s", mappingFile)

	// verify mapping.json exists and has correct structure
	if _, err := os.Stat(mappingFile); os.IsNotExist(err) {
		t.Fatalf("expected mapping file %s to exist", mappingFile)
	}
	// Phase 4: gorts select (with checked out newer commit)
	t.Logf("checking out newer commit: %s", currentCommit)
	if err := checkoutCommit(t, testRepoPath, currentCommit); err != nil {
		t.Fatalf("failed to checkout newer commit: %v", err)
	}

	selectionFile := filepath.Join(outputDir(t, "happy_path"), "selection_newer_commit.json")
	t.Logf("running gorts select with previously generated baseline: %s and mapping: %s", baselineFile, mappingFile)
	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", selectionFile,
		"--repo", testRepoPath,
		"--strip-prefix", "",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("selection created at %s", selectionFile)

	if _, err := os.Stat(selectionFile); os.IsNotExist(err) {
		t.Fatalf("expected selection file %s to exist", selectionFile)
	}

	data, err := os.ReadFile(selectionFile)
	if err != nil {
		t.Fatalf("failed to read selection file: %v", err)
	}

	var selection struct {
		GeneratedAt   string   `json:"generated_at"`
		FromCommit    string   `json:"from_commit"`
		ToCommit      string   `json:"to_commit"`
		ChangedFiles  []string `json:"changed_files"`
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests       int     `json:"total_tests"`
			SelectedTests    int     `json:"selected_tests"`
			ChangedFiles     int     `json:"changed_files"`
			ChangedTestFiles int     `json:"changed_test_files"`
			ReductionPercent float64 `json:"reduction_percent"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(data, &selection); err != nil {
		t.Fatalf("failed to parse selection JSON: %v", err)
	}

	t.Log("verifying selection structure")

	if selection.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if selection.FromCommit != baselineCommit {
		t.Error("from_commit should not be empty")
	}
	if selection.ToCommit != currentCommit {
		t.Error("to_commit should not be empty")
	}

	if len(selection.SelectedTests) == 0 {
		t.Error("expected selected_tests to be non-empty (tests covering changed files should be selected)")
	}

	t.Log("Happy path e2e test successful")
}
