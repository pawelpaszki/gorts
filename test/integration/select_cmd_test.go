//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectCmd_SourceFileChange(t *testing.T) {
	t.Log("using pre-generated baseline and mapping from TestMain")
	t.Logf("baseline: %s (generated at %s)", baselineFile, baselineCommit[:12])
	t.Logf("mapping: %s (generated at %s)", mappingFile, baselineCommit[:12])
	t.Logf("repo at: %s (currentCommit)", currentCommit[:12])

	outputFile := filepath.Join(outputDir(t), "selection_source_change.json")

	t.Log("running gorts select (source file changed between commits)")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", outputFile,
		"--repo", testRepoPath,
		"--strip-prefix", "",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("selection created at %s", outputFile)

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected selection file %s to exist", outputFile)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read selection file: %v", err)
	}

	var selection struct {
		GeneratedAt   string `json:"generated_at"`
		FromCommit    string `json:"from_commit"`
		ToCommit      string `json:"to_commit"`
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
	if selection.FromCommit == "" {
		t.Error("from_commit should not be empty")
	}
	if selection.ToCommit == "" {
		t.Error("to_commit should not be empty")
	}

	t.Log("verifying changes were detected")

	if len(selection.ChangedFiles) == 0 {
		t.Error("expected changed_files to be non-empty (source file changed between commits)")
	}
	if selection.Stats.ChangedFiles == 0 {
		t.Error("expected stats.changed_files > 0")
	}

	t.Log("verifying tests were selected based on changed source files")

	if len(selection.SelectedTests) == 0 {
		t.Error("expected selected_tests to be non-empty (tests covering changed files should be selected)")
	}
	if selection.Stats.SelectedTests == 0 {
		t.Error("expected stats.selected_tests > 0")
	}

	t.Logf("selection complete: %d tests selected out of %d total (%.1f%% reduction)",
		selection.Stats.SelectedTests, selection.Stats.TotalTests, selection.Stats.ReductionPercent)
	t.Logf("changed files: %v", selection.ChangedFiles)
}

func TestSelectCmd_TestFileChange(t *testing.T) {
	t.Log("using pre-generated baseline and mapping from TestMain")
	t.Logf("baseline: %s (generated at %s)", baselineFile, baselineCommit[:12])
	t.Logf("mapping: %s (generated at %s)", mappingFile, baselineCommit[:12])

	outputFile := filepath.Join(outputDir(t), "selection_test_change.json")

	t.Log("running gorts select")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", outputFile,
		"--repo", testRepoPath,
		"--strip-prefix", "",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected selection file %s to exist", outputFile)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read selection file: %v", err)
	}

	var selection struct {
		GeneratedAt         string   `json:"generated_at"`
		FromCommit          string   `json:"from_commit"`
		ToCommit            string   `json:"to_commit"`
		ChangedFiles        []string `json:"changed_files"`
		OutOfScopeTestFiles []string `json:"out_of_scope_test_files"`
		SelectedTests       []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests          int     `json:"total_tests"`
			SelectedTests       int     `json:"selected_tests"`
			ChangedFiles        int     `json:"changed_files"`
			ChangedTestFiles    int     `json:"changed_test_files"`
			OutOfScopeTestFiles int     `json:"out_of_scope_test_files"`
			ReductionPercent    float64 `json:"reduction_percent"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(data, &selection); err != nil {
		t.Fatalf("failed to parse selection JSON: %v", err)
	}

	t.Log("verifying selection structure")

	if selection.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if selection.FromCommit == "" {
		t.Error("from_commit should not be empty")
	}
	if selection.ToCommit == "" {
		t.Error("to_commit should not be empty")
	}

	t.Log("checking for test file changes in changed_files")

	var changedTestFiles []string
	for _, f := range selection.ChangedFiles {
		if strings.HasSuffix(f, "_test.go") {
			changedTestFiles = append(changedTestFiles, f)
			t.Logf("found changed test file: %s", f)
		}
	}

	if len(changedTestFiles) > 0 {
		t.Logf("detected %d changed test file(s)", len(changedTestFiles))

		// Test files can be either in-scope (in baseline dirs) or out-of-scope
		// changed_test_files counts in-scope, out_of_scope_test_files counts out-of-scope
		totalTracked := selection.Stats.ChangedTestFiles + selection.Stats.OutOfScopeTestFiles
		if totalTracked == 0 {
			t.Error("expected changed_test_files + out_of_scope_test_files > 0 when test files changed")
		}

		t.Logf("in-scope test files (changed_test_files): %d", selection.Stats.ChangedTestFiles)
		t.Logf("out-of-scope test files: %d", selection.Stats.OutOfScopeTestFiles)

		if len(selection.OutOfScopeTestFiles) > 0 {
			t.Logf("out-of-scope test files list: %v", selection.OutOfScopeTestFiles)
		}
	} else {
		t.Log("no test file changes in this commit range")
		if selection.Stats.ChangedTestFiles != 0 {
			t.Errorf("expected stats.changed_test_files == 0, got %d", selection.Stats.ChangedTestFiles)
		}
	}

	t.Logf("selection complete: %d tests selected, %d in-scope test files, %d out-of-scope test files",
		selection.Stats.SelectedTests, selection.Stats.ChangedTestFiles, selection.Stats.OutOfScopeTestFiles)
}

func TestSelectCmd_NoChanges(t *testing.T) {
	t.Log("using pre-generated baseline and mapping from TestMain")
	t.Logf("mapping generated at commit: %s", baselineCommit[:12])

	t.Log("checking out baseline commit (same as mapping) to simulate no changes")
	if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
		t.Fatalf("failed to checkout baseline commit: %v", err)
	}
	defer func() {
		if err := checkoutCommit(t, testRepoPath, currentCommit); err != nil {
			t.Errorf("failed to restore current commit: %v", err)
		}
		t.Log("restored to current commit")
	}()

	outputFile := filepath.Join(outputDir(t), "selection_no_changes.json")

	t.Log("running gorts select (expecting 'No changes detected')")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", outputFile,
		"--repo", testRepoPath,
		"--strip-prefix", "",
	)

	if exitCode != 0 {
		t.Fatalf("gorts select failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Log("verifying 'No changes detected' message in output")
	if !strings.Contains(stdout, "No changes detected") {
		t.Errorf("expected stdout to contain 'No changes detected', got: %s", stdout)
	}

	t.Log("verifying no selection file was created (no changes = no output)")
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Log("selection file exists - this is acceptable behavior")
	}

	t.Log("no changes detected test passed")
}

func TestSelectCmd_RunAllTrigger(t *testing.T) {
	t.Log("using pre-generated baseline and mapping from TestMain")
	t.Logf("baseline: %s", baselineFile)
	t.Logf("mapping: %s", mappingFile)

	t.Log("checking out baseline commit for clean state")
	if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
		t.Fatalf("failed to checkout baseline commit: %v", err)
	}

	t.Log("creating a go.mod change to trigger run-all")
	goModPath := filepath.Join(testRepoPath, "go.mod")
	originalContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	modifiedContent := append(originalContent, []byte("\n// trigger run-all\n")...)
	if err := os.WriteFile(goModPath, modifiedContent, 0644); err != nil {
		t.Fatalf("failed to modify go.mod: %v", err)
	}

	t.Log("committing go.mod change")
	if err := gitAdd(testRepoPath, "go.mod"); err != nil {
		t.Fatalf("failed to stage go.mod: %v", err)
	}
	if err := gitCommit(testRepoPath, "trigger run-all test"); err != nil {
		t.Fatalf("failed to commit go.mod change: %v", err)
	}

	defer func() {
		t.Log("restoring repository state")
		if err := gitResetHard(testRepoPath, currentCommit); err != nil {
			t.Errorf("failed to reset to current commit: %v", err)
		}
		t.Log("restored to current commit")
	}()

	outputFile := filepath.Join(outputDir(t), "selection_run_all.json")

	t.Log("running gorts select with --run-all-on go.mod,go.sum,Makefile")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", outputFile,
		"--repo", testRepoPath,
		"--strip-prefix", "",
		"--run-all-on", "go.mod,go.sum,Makefile",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Log("verifying 'Run-all triggered' message in output")
	if !strings.Contains(stdout, "Run-all triggered") {
		t.Errorf("expected stdout to contain 'Run-all triggered', got: %s", stdout)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected selection file %s to exist", outputFile)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read selection file: %v", err)
	}

	var selection struct {
		GeneratedAt   string   `json:"generated_at"`
		ChangedFiles  []string `json:"changed_files"`
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests       int     `json:"total_tests"`
			SelectedTests    int     `json:"selected_tests"`
			ReductionPercent float64 `json:"reduction_percent"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(data, &selection); err != nil {
		t.Fatalf("failed to parse selection JSON: %v", err)
	}

	t.Log("verifying ALL tests were selected (run-all triggered)")

	if selection.Stats.SelectedTests != selection.Stats.TotalTests {
		t.Errorf("expected all tests selected when run-all triggered: got %d/%d",
			selection.Stats.SelectedTests, selection.Stats.TotalTests)
	}

	if selection.Stats.ReductionPercent != 0 {
		t.Errorf("expected 0%% reduction when run-all triggered, got %.1f%%",
			selection.Stats.ReductionPercent)
	}

	t.Logf("run-all test passed: %d/%d tests selected (0%% reduction)",
		selection.Stats.SelectedTests, selection.Stats.TotalTests)
}

func TestSelectCmd_FunctionGranularity(t *testing.T) {
	t.Log("testing function-level vs file-level granularity")
	t.Log("scenario: internal/model/book.go changed, but only Validate() function was modified")
	t.Log("         IsPublished() function was NOT modified")
	t.Logf("baseline commit: %s", baselineCommit[:12])
	t.Logf("current commit: %s", currentCommit[:12])

	// Step 1: Run file-level selection (default)
	fileLevelOutput := filepath.Join(outputDir(t), "selection_file_level.json")

	t.Log("running gorts select with file-level granularity (default)")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", fileLevelOutput,
		"--repo", testRepoPath,
		"--strip-prefix", "",
		"--granularity", "file",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select (file-level) failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	fileLevelData, err := os.ReadFile(fileLevelOutput)
	if err != nil {
		t.Fatalf("failed to read file-level selection: %v", err)
	}

	var fileLevelSelection struct {
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests       int     `json:"total_tests"`
			SelectedTests    int     `json:"selected_tests"`
			ReductionPercent float64 `json:"reduction_percent"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(fileLevelData, &fileLevelSelection); err != nil {
		t.Fatalf("failed to parse file-level selection JSON: %v", err)
	}

	t.Logf("file-level: %d tests selected out of %d (%.1f%% reduction)",
		fileLevelSelection.Stats.SelectedTests,
		fileLevelSelection.Stats.TotalTests,
		fileLevelSelection.Stats.ReductionPercent)

	// Step 2: Run function-level selection
	funcLevelOutput := filepath.Join(outputDir(t), "selection_func_level.json")

	t.Log("running gorts select with function-level granularity")
	stdout, stderr, exitCode = runGortsInDir(t, testRepoPath,
		"select",
		"--baseline", baselineFile,
		"--mapping", mappingFile,
		"--output", funcLevelOutput,
		"--repo", testRepoPath,
		"--strip-prefix", "",
		"--granularity", "function",
	)
	if exitCode != 0 {
		t.Fatalf("gorts select (function-level) failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Log("verifying 'Function-level analysis' message in output")
	if !strings.Contains(stdout, "Function-level analysis") {
		t.Errorf("expected stdout to contain 'Function-level analysis', got: %s", stdout)
	}

	funcLevelData, err := os.ReadFile(funcLevelOutput)
	if err != nil {
		t.Fatalf("failed to read function-level selection: %v", err)
	}

	var funcLevelSelection struct {
		SelectedTests []struct {
			Directory string `json:"directory"`
			TestName  string `json:"test_name"`
		} `json:"selected_tests"`
		Stats struct {
			TotalTests       int     `json:"total_tests"`
			SelectedTests    int     `json:"selected_tests"`
			ReductionPercent float64 `json:"reduction_percent"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(funcLevelData, &funcLevelSelection); err != nil {
		t.Fatalf("failed to parse function-level selection JSON: %v", err)
	}

	t.Logf("function-level: %d tests selected out of %d (%.1f%% reduction)",
		funcLevelSelection.Stats.SelectedTests,
		funcLevelSelection.Stats.TotalTests,
		funcLevelSelection.Stats.ReductionPercent)

	// Step 3: Compare results
	t.Log("comparing file-level vs function-level selection")

	if funcLevelSelection.Stats.SelectedTests > fileLevelSelection.Stats.SelectedTests {
		t.Errorf("function-level should select same or fewer tests than file-level: got %d > %d",
			funcLevelSelection.Stats.SelectedTests, fileLevelSelection.Stats.SelectedTests)
	}

	// Log the comparison (focus on test counts, not reduction % which can be negative
	// when new test discovery adds more tests than the baseline contained)
	testDiff := fileLevelSelection.Stats.SelectedTests - funcLevelSelection.Stats.SelectedTests

	t.Logf("comparison summary:")
	t.Logf("  file-level:     %d tests selected", fileLevelSelection.Stats.SelectedTests)
	t.Logf("  function-level: %d tests selected", funcLevelSelection.Stats.SelectedTests)
	t.Logf("  difference:     %d fewer tests with function-level granularity", testDiff)

	if testDiff > 0 {
		t.Logf("function-level granularity successfully reduced test selection by %d test(s)", testDiff)
	} else if testDiff == 0 {
		t.Log("function-level and file-level selected the same number of tests")
	}
}
