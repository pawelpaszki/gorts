//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestE2E_Selection_Safety validates the complete gorts workflow:
// gorts tests -> gorts baseline -> gorts mapping -> gorts select
//
// This test verifies that artificial breaking changes get picked
// up by select function
func TestE2E_Selection_Safety(t *testing.T) {
	// Restore repo to baselineCommit after test for other tests
	defer func() {
		// Hard reset to discard the test commit we created
		resetCmd := exec.Command("git", "reset", "--hard", baselineCommit)
		resetCmd.Dir = testRepoPath
		if output, err := resetCmd.CombinedOutput(); err != nil {
			t.Errorf("failed to reset repo: %v\noutput: %s", err, output)
		}
	}()

	t.Log("using pre-built test binary from TestMain")
	t.Logf("test binary: %s", testBinaryPath)

	t.Logf("checking out latest commit: %s", currentCommit)
	if err := checkoutCommit(t, testRepoPath, currentCommit); err != nil {
		t.Fatalf("failed to checkout newer commit: %v", err)
	}

	// Phase 1: gorts tests
	t.Log("running gorts tests with single directory: ./test/e2e")

	manifestFile := filepath.Join(outputDir(t, "selection_safety"), "manifest.json")

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
	coverageDir := filepath.Join(outputDir(t, "selection_safety"), "coverage_test")
	baselineFile := filepath.Join(outputDir(t, "selection_safety"), "baseline_test.json")

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
	mappingFile := filepath.Join(outputDir(t, "selection_safety"), "mapping_file_level.json")

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

	// Phase 4: Introduce breaking changes by copying fixture file
	t.Log("introducing breaking changes to internal/model/book.go")

	// Find the fixture file (relative to this test file)
	_, thisFile, _, _ := runtime.Caller(0)
	fixtureFile := filepath.Join(filepath.Dir(thisFile), "fixtures", "breaking_book.txt")

	// Read the breaking changes fixture
	breakingContent, err := os.ReadFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to read fixture file %s: %v", fixtureFile, err)
	}

	// Overwrite the book.go file with breaking changes
	targetFile := filepath.Join(testRepoPath, "internal", "model", "book.go")
	if err := os.WriteFile(targetFile, breakingContent, 0644); err != nil {
		t.Fatalf("failed to write breaking changes to %s: %v", targetFile, err)
	}
	t.Logf("copied breaking changes to %s", targetFile)

	// Git add and commit the changed file so git diff sees it
	addCmd := exec.Command("git", "add", "internal/model/book.go")
	addCmd.Dir = testRepoPath
	if output, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git add: %v\noutput: %s", err, output)
	}

	commitCmd := exec.Command("git", "commit", "-m", "test: introduce breaking change for safety validation")
	commitCmd.Dir = testRepoPath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git commit: %v\noutput: %s", err, output)
	}
	t.Log("committed breaking changes")

	// Phase 5: Run gorts select - should detect the changed file and select affected tests
	t.Log("running gorts select to detect breaking changes")
	selectionFile := filepath.Join(outputDir(t, "selection_safety"), "selection.json")

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

	// Phase 6: Verify the affected tests were selected
	selectionData, err := os.ReadFile(selectionFile)
	if err != nil {
		t.Fatalf("failed to read selection file: %v", err)
	}

	var selection struct {
		ChangedFiles  []string `json:"changed_files"`
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests    int `json:"total_tests"`
			SelectedTests int `json:"selected_tests"`
			ChangedFiles  int `json:"changed_files"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(selectionData, &selection); err != nil {
		t.Fatalf("failed to parse selection JSON: %v", err)
	}

	// Verify book.go was detected as changed
	bookGoChanged := false
	for _, f := range selection.ChangedFiles {
		if f == "internal/model/book.go" {
			bookGoChanged = true
			break
		}
	}
	if !bookGoChanged {
		t.Errorf("expected internal/model/book.go in changed_files, got: %v", selection.ChangedFiles)
	}

	// Verify tests were selected (book.go should be covered by tests)
	if selection.Stats.SelectedTests == 0 {
		t.Error("expected at least one test to be selected for the breaking change")
	}

	t.Logf("safety validation successful:")
	t.Logf("  changed files: %d", selection.Stats.ChangedFiles)
	t.Logf("  selected tests: %d (out of %d total)", selection.Stats.SelectedTests, selection.Stats.TotalTests)
	t.Logf("  breaking change in book.go correctly triggered test selection")
}
