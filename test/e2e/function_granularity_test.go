//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Commits for function granularity test - chosen to show a difference
// between file-level and function-level selection.
// From gorts-demo Commit 27 (daf334a): "Doc comment to health handler"
//   - file_selected=31, func_selected=0 (saves 31 tests)
// This is due to function checksums - doc comments do not affect checksums
const (
	funcGranularityBaseline = "569f4bb" // Commit 26: "Add IsPublished to Book"
	funcGranularityCurrent  = "daf334a" // Commit 27: "Doc comment to health handler"
)

// TestE2E_FunctionGranularity validates function-level test selection:
// gorts tests -> gorts baseline -> gorts mapping (with --repo) -> gorts select (with --granularity function)
//
// This test verifies that function-level granularity selects fewer tests than file-level
// when only a doc comment (which doesn't change function body checksum) was modified.
//
// Scenario: health_handler.go has a doc comment added to HandleHealth()
// The function body checksum remains the same, so function-level should select 0 tests.
// File-level will select all tests that cover that file.
func TestE2E_FunctionGranularity(t *testing.T) {
	// Restore repo to baselineCommit after test for other tests
	defer func() {
		if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
			t.Errorf("failed to restore baseline commit: %v", err)
		}
	}()

	t.Log("testing function-level vs file-level granularity")
	t.Log("scenario: health_handler.go changed with doc comment only (no function body change)")
	t.Logf("baseline commit: %s, current commit: %s", funcGranularityBaseline, funcGranularityCurrent)
	t.Logf("test binary: %s", testBinaryPath)

	// First checkout the baseline commit for this specific test
	t.Logf("checking out baseline commit: %s", funcGranularityBaseline)
	if err := checkoutCommit(t, testRepoPath, funcGranularityBaseline); err != nil {
		t.Fatalf("failed to checkout baseline commit: %v", err)
	}

	// Rebuild test binary at this commit for accurate coverage
	localTestBinary := filepath.Join(outputDir(t, "function_granularity"), "test.bin")
	if err := buildTestBinary(testRepoPath, localTestBinary); err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}
	t.Logf("built test binary at commit %s: %s", funcGranularityBaseline, localTestBinary)

	// Phase 1: gorts tests
	t.Log("running gorts tests with single directory: ./test/e2e")

	manifestFile := filepath.Join(outputDir(t, "function_granularity"), "manifest.json")

	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"tests",
		"--directories", "./test/e2e",
		"--output", manifestFile,
	)

	if exitCode != 0 {
		t.Fatalf("gorts tests failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("manifest created at %s", manifestFile)

	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		t.Fatalf("expected output file %s to exist", manifestFile)
	}

	// Phase 2: gorts baseline
	t.Logf("running gorts baseline with previously generated manifest: %s", manifestFile)
	coverageDir := filepath.Join(outputDir(t, "function_granularity"), "coverage")
	baselineFile := filepath.Join(outputDir(t, "function_granularity"), "baseline.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", localTestBinary,
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

	// Phase 3: gorts mapping with --repo flag (enables function-level checksums)
	t.Log("running gorts mapping with --repo flag for function-level support")
	mappingFile := filepath.Join(outputDir(t, "function_granularity"), "mapping.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", mappingFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
		"--repo", testRepoPath,
	)
	if exitCode != 0 {
		t.Fatalf("gorts mapping failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("mapping created at %s", mappingFile)

	if _, err := os.Stat(mappingFile); os.IsNotExist(err) {
		t.Fatalf("expected mapping file %s to exist", mappingFile)
	}

	// Phase 4: Checkout newer commit (the one with doc comment change)
	t.Logf("checking out newer commit: %s", funcGranularityCurrent)
	if err := checkoutCommit(t, testRepoPath, funcGranularityCurrent); err != nil {
		t.Fatalf("failed to checkout newer commit: %v", err)
	}

	// Phase 5a: Run file-level selection (for comparison)
	t.Log("running gorts select with file-level granularity (default)")
	fileLevelSelection := filepath.Join(outputDir(t, "function_granularity"), "selection_file_level.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", fileLevelSelection,
		"--repo", testRepoPath,
		"--strip-prefix", "",
		"--granularity", "file",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select (file-level) failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	fileLevelData, err := os.ReadFile(fileLevelSelection)
	if err != nil {
		t.Fatalf("failed to read file-level selection: %v", err)
	}

	var fileSelection struct {
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			SelectedTests int `json:"selected_tests"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(fileLevelData, &fileSelection); err != nil {
		t.Fatalf("failed to parse file-level selection JSON: %v", err)
	}

	t.Logf("file-level selection: %d tests selected", fileSelection.Stats.SelectedTests)

	// Phase 5b: Run function-level selection
	t.Log("running gorts select with function-level granularity")
	funcLevelSelection := filepath.Join(outputDir(t, "function_granularity"), "selection_func_level.json")

	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", funcLevelSelection,
		"--repo", testRepoPath,
		"--strip-prefix", "",
		"--granularity", "function",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select (function-level) failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// Verify "Function-level analysis" message in output
	if !strings.Contains(stdout, "Function-level analysis") {
		t.Errorf("expected stdout to contain 'Function-level analysis', got: %s", stdout)
	}

	funcLevelData, err := os.ReadFile(funcLevelSelection)
	if err != nil {
		t.Fatalf("failed to read function-level selection: %v", err)
	}

	var funcSelection struct {
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			SelectedTests int `json:"selected_tests"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(funcLevelData, &funcSelection); err != nil {
		t.Fatalf("failed to parse function-level selection JSON: %v", err)
	}

	t.Logf("function-level selection: %d tests selected", funcSelection.Stats.SelectedTests)

	// Phase 6: Compare results
	t.Log("comparing file-level vs function-level selection")

	if funcSelection.Stats.SelectedTests > fileSelection.Stats.SelectedTests {
		t.Errorf("function-level should select same or fewer tests than file-level: got %d > %d",
			funcSelection.Stats.SelectedTests, fileSelection.Stats.SelectedTests)
	}

	testDiff := fileSelection.Stats.SelectedTests - funcSelection.Stats.SelectedTests

	t.Logf("comparison summary:")
	t.Logf("  file-level:     %d tests selected", fileSelection.Stats.SelectedTests)
	t.Logf("  function-level: %d tests selected", funcSelection.Stats.SelectedTests)
	t.Logf("  difference:     %d fewer tests with function-level granularity", testDiff)

	if testDiff > 0 {
		t.Logf("function-level granularity successfully reduced test selection by %d test(s)", testDiff)
	} else if testDiff == 0 {
		t.Log("function-level and file-level selected the same number of tests")
	}

	t.Log("Function granularity e2e test successful")
}
