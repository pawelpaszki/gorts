//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMappingCmd_FileLevelMapping(t *testing.T) {
	t.Log("using pre-generated baseline from TestMain")
	t.Logf("baseline: %s", baselineFile)

	// run mapping (file-level only, no --repo flag)
	t.Log("running gorts mapping (file-level)")

	outputFile := filepath.Join(outputDir(t), "mapping_file_level.json")

	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", outputFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
	)
	if exitCode != 0 {
		t.Fatalf("gorts mapping failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("mapping created at %s", outputFile)

	// verify mapping.json exists and has correct structure
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected mapping file %s to exist", outputFile)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read mapping file: %v", err)
	}

	var mapping struct {
		GeneratedAt string              `json:"generated_at"`
		CommitSHA   string              `json:"commit_sha"`
		FileToTests map[string][]string `json:"file_to_tests"`
		TestToFiles map[string][]string `json:"test_to_files"`
		Stats       struct {
			TotalTests      int     `json:"total_tests"`
			TotalFiles      int     `json:"total_files"`
			AvgFilesPerTest float64 `json:"avg_files_per_test"`
			AvgTestsPerFile float64 `json:"avg_tests_per_file"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(data, &mapping); err != nil {
		t.Fatalf("failed to parse mapping JSON: %v", err)
	}

	t.Log("verifying mapping structure")

	if mapping.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if mapping.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}
	if len(mapping.FileToTests) == 0 {
		t.Error("file_to_tests should not be empty")
	}
	if len(mapping.TestToFiles) == 0 {
		t.Error("test_to_files should not be empty")
	}
	if mapping.Stats.TotalTests == 0 {
		t.Error("stats.total_tests should be > 0")
	}
	if mapping.Stats.TotalFiles == 0 {
		t.Error("stats.total_files should be > 0")
	}

	t.Logf("mapping complete: %d tests covering %d files (avg %.1f files/test)",
		mapping.Stats.TotalTests, mapping.Stats.TotalFiles, mapping.Stats.AvgFilesPerTest)
}

func TestMappingCmd_FunctionLevelMapping(t *testing.T) {
	t.Log("using pre-generated baseline from TestMain")
	t.Logf("baseline: %s", baselineFile)

	// temporarily checkout baselineCommit (--repo requires repo at same commit as baseline)
	t.Log("checking out baseline commit for --repo flag compatibility")
	if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
		t.Fatalf("failed to checkout baseline commit: %v", err)
	}
	defer func() {
		// restore to currentCommit after test
		if err := checkoutCommit(t, testRepoPath, currentCommit); err != nil {
			t.Errorf("failed to restore current commit: %v", err)
		}
		t.Log("restored to current commit")
	}()

	// run mapping with --repo flag (enables function-level mapping with checksums)
	t.Log("running gorts mapping (function-level with --repo)")

	outputFile := filepath.Join(outputDir(t), "mapping_func_level.json")

	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", outputFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
		"--repo", testRepoPath,
	)
	if exitCode != 0 {
		t.Fatalf("gorts mapping failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("mapping created at %s", outputFile)

	// verify mapping.json exists and has function-level data
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected mapping file %s to exist", outputFile)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read mapping file: %v", err)
	}

	var mapping struct {
		GeneratedAt       string              `json:"generated_at"`
		CommitSHA         string              `json:"commit_sha"`
		FileToTests       map[string][]string `json:"file_to_tests"`
		TestToFiles       map[string][]string `json:"test_to_files"`
		FunctionToTests   map[string][]string `json:"function_to_tests"`
		TestToFunctions   map[string][]string `json:"test_to_functions"`
		FunctionChecksums map[string]string   `json:"function_checksums"`
		Stats             struct {
			TotalTests          int     `json:"total_tests"`
			TotalFiles          int     `json:"total_files"`
			TotalFunctions      int     `json:"total_functions"`
			AvgFilesPerTest     float64 `json:"avg_files_per_test"`
			AvgTestsPerFile     float64 `json:"avg_tests_per_file"`
			AvgFunctionsPerTest float64 `json:"avg_functions_per_test"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(data, &mapping); err != nil {
		t.Fatalf("failed to parse mapping JSON: %v", err)
	}

	t.Log("verifying mapping structure (file-level)")

	if mapping.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if mapping.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}
	if len(mapping.FileToTests) == 0 {
		t.Error("file_to_tests should not be empty")
	}
	if len(mapping.TestToFiles) == 0 {
		t.Error("test_to_files should not be empty")
	}

	t.Log("verifying mapping structure (function-level)")

	if len(mapping.FunctionToTests) == 0 {
		t.Error("function_to_tests should not be empty (--repo was provided)")
	}
	if len(mapping.TestToFunctions) == 0 {
		t.Error("test_to_functions should not be empty (--repo was provided)")
	}
	if len(mapping.FunctionChecksums) == 0 {
		t.Error("function_checksums should not be empty (--repo was provided)")
	}
	if mapping.Stats.TotalFunctions == 0 {
		t.Error("stats.total_functions should be > 0")
	}

	t.Logf("mapping complete: %d tests, %d files, %d functions (avg %.1f functions/test)",
		mapping.Stats.TotalTests, mapping.Stats.TotalFiles,
		mapping.Stats.TotalFunctions, mapping.Stats.AvgFunctionsPerTest)
}

func TestMappingCmd_MissingBaseline(t *testing.T) {
	t.Log("running gorts mapping with nonexistent baseline (expecting failure)")

	outputFile := filepath.Join(outputDir(t), "mapping_missing_baseline.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", "/nonexistent/baseline.json",
		"--output", outputFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing baseline")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for missing baseline")
}

func TestMappingCmd_MissingModuleFlag(t *testing.T) {
	t.Log("running gorts mapping without --module flag (expecting failure)")

	outputFile := filepath.Join(outputDir(t), "mapping_missing_module.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", outputFile,
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing --module flag")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for missing --module flag")
}

func TestMappingCmd_DirtyRepo(t *testing.T) {
	t.Log("running gorts mapping with dirty repo (expecting failure)")

	// checkout baselineCommit (--repo requires repo at same commit as baseline)
	t.Log("checking out baseline commit for --repo flag compatibility")
	if err := checkoutCommit(t, testRepoPath, baselineCommit); err != nil {
		t.Fatalf("failed to checkout baseline commit: %v", err)
	}
	defer func() {
		// restore to currentCommit after test
		if err := checkoutCommit(t, testRepoPath, currentCommit); err != nil {
			t.Errorf("failed to restore current commit: %v", err)
		}
		t.Log("restored to current commit")
	}()

	// create an uncommitted .go file to make the repo dirty
	// (VerifyCleanRepo only checks for .go, go.mod, go.sum files)
	dirtyFile := filepath.Join(testRepoPath, "dirty_test_file.go")
	if err := os.WriteFile(dirtyFile, []byte("package dirty\n"), 0644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}
	// clean up the dirty file after test
	defer func() {
		os.Remove(dirtyFile)
		t.Log("cleaned up dirty file")
	}()

	t.Log("created uncommitted .go file to make repo dirty")

	outputFile := filepath.Join(outputDir(t), "mapping_dirty.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"mapping",
		"--baseline", baselineFile,
		"--output", outputFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
		"--repo", testRepoPath,
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for dirty repo")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for dirty repo")
}
