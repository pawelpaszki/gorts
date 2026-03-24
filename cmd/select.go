package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pawelpaszki/gorts/internal/coverage"
	"github.com/pawelpaszki/gorts/internal/exec"
	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/spf13/cobra"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select tests using gorts",
	Long:  `Select tests to execute based on recorded baseline and changes between revisions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		baselinePath, _ := cmd.Flags().GetString("baseline")
		mappingPath, _ := cmd.Flags().GetString("mapping")
		outputPath, _ := cmd.Flags().GetString("output")
		repoPath, _ := cmd.Flags().GetString("repo")
		stripPrefix, _ := cmd.Flags().GetString("strip-prefix")
		granularity, _ := cmd.Flags().GetString("granularity")

		// Load baseline (for directory info and validation)
		baseline, err := jsonutil.LoadBaseline(baselinePath)
		if err != nil {
			return fmt.Errorf("failed to load baseline: %w", err)
		}

		// Load mapping
		mapping, err := jsonutil.LoadMapping(mappingPath)
		if err != nil {
			return fmt.Errorf("failed to load mapping: %w", err)
		}

		// Get current commit SHA from repo
		currentCommit, _, err := exec.Run(repoPath, "git", "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("failed to get current commit: %w", err)
		}
		currentCommit = strings.TrimSpace(currentCommit)

		if baseline.CommitSHA != mapping.CommitSHA {
			fmt.Printf("[Warn] Baseline commit (%s) differs from mapping commit (%s)\n",
				baseline.CommitSHA[:12], mapping.CommitSHA[:12])
		}

		// Check if commits are the same
		if mapping.CommitSHA == currentCommit {
			fmt.Println("==================================================")
			fmt.Println("No changes detected!")
			fmt.Printf("  Baseline commit: %s\n", mapping.CommitSHA)
			fmt.Printf("  Current commit:  %s\n", currentCommit)
			fmt.Println("  Recommendation: No tests need to be run")
			fmt.Println("==================================================")
			return nil
		}

		// Get ALL changed files between baseline commit and current (including non-.go files)
		allChangedFiles, err := getAllChangedFiles(repoPath, mapping.CommitSHA, currentCommit, stripPrefix)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		if len(allChangedFiles) == 0 {
			fmt.Println("No files changed between commits")
			return nil
		}

		// Check run-all patterns BEFORE filtering to .go files
		runAllPatterns, _ := cmd.Flags().GetStringSlice("run-all-on")
		runAll, triggerFile := model.CheckRunAllTrigger(allChangedFiles, runAllPatterns)

		// Filter to .go files for normal selection
		var changedFiles []string
		for _, f := range allChangedFiles {
			if strings.HasSuffix(f, ".go") {
				changedFiles = append(changedFiles, f)
			}
		}

		// If no .go files changed and no run-all trigger, nothing to do
		if !runAll && len(changedFiles) == 0 {
			fmt.Println("No source files changed (only non-code files modified)")
			return nil
		}

		// Initialize to empty slice (not nil) for cleaner JSON output
		selectedTests := []model.SelectedTest{}
		var changedTestFilesCount int
		var outOfScopeTestFilesCount int
		var outOfScopeTestFiles []string
		var newTestsCount int

		// Build set of baseline directories for scope checking
		baselineDirs := make(map[string]bool)
		for _, suite := range baseline.TestSuiteResults {
			baselineDirs[suite.Directory] = true
		}

		if runAll {
			// Run all tests
			fmt.Printf("[Info] Run-all triggered by: %s\n", triggerFile)

			// Get all tests from mapping
			for qualifiedName := range mapping.TestToFiles {
				dir, testName := model.ParseQualifiedTest(qualifiedName)
				selectedTests = append(selectedTests, model.SelectedTest{
					Directory: dir,
					TestName:  testName,
				})
			}
		} else {
			// Separate changed files into source files and test files
			var sourceFiles []string
			var inScopeTestFiles []string
			for _, file := range changedFiles {
				if strings.HasSuffix(file, "_test.go") {
					// Check if this test file is in baseline scope
					pkgDir := filepath.Dir(file)
					if baselineDirs[pkgDir] {
						inScopeTestFiles = append(inScopeTestFiles, file)
					} else {
						outOfScopeTestFiles = append(outOfScopeTestFiles, file)
					}
				} else {
					sourceFiles = append(sourceFiles, file)
				}
			}
			changedTestFilesCount = len(inScopeTestFiles)
			outOfScopeTestFilesCount = len(outOfScopeTestFiles)

			// Select tests based on changed files
			selectedTestsMap := make(map[string]bool) // qualifiedName -> selected

			// Coverage-based selection for source files
			if granularity == "function" && len(mapping.FunctionChecksums) > 0 {
				// Function-level selection: compare checksums to find changed functions
				changedFunctions := findChangedFunctions(repoPath, sourceFiles, mapping.FunctionChecksums)
				fmt.Printf("[Info] Function-level analysis: %d changed functions detected\n", len(changedFunctions))

				for _, qualifiedFunc := range changedFunctions {
					if tests, ok := mapping.FunctionToTests[qualifiedFunc]; ok {
						for _, qualifiedName := range tests {
							selectedTestsMap[qualifiedName] = true
						}
					}
				}
			} else {
				// File-level selection (default)
				for _, file := range sourceFiles {
					if tests, ok := mapping.FileToTests[file]; ok {
						for _, qualifiedName := range tests {
							selectedTestsMap[qualifiedName] = true
						}
					}
				}
			}

			// For changed _test.go files (in scope), select ALL tests in that package
			for _, testFile := range inScopeTestFiles {
				pkgDir := filepath.Dir(testFile)
				for qualifiedName := range mapping.TestToFiles {
					dir, _ := model.ParseQualifiedTest(qualifiedName)
					if dir == pkgDir {
						selectedTestsMap[qualifiedName] = true
					}
				}
			}

			// Discover new tests not in the baseline mapping
			newTests, err := discoverNewTests(repoPath, mapping, baseline)
			if err != nil {
				fmt.Printf("[Warn] Failed to discover new tests: %v\n", err)
			} else {
				for _, qualifiedName := range newTests {
					selectedTestsMap[qualifiedName] = true
				}
				newTestsCount = len(newTests)
				if newTestsCount > 0 {
					fmt.Printf("[Info] Discovered %d new test(s) not in baseline\n", newTestsCount)
				}
			}

			// Build selected tests slice
			for qualifiedName := range selectedTestsMap {
				dir, testName := model.ParseQualifiedTest(qualifiedName)
				selectedTests = append(selectedTests, model.SelectedTest{
					Directory: dir,
					TestName:  testName,
				})
			}
		}

		// Calculate stats
		totalTests := mapping.Stats.TotalTests
		selectedCount := len(selectedTests)
		reductionPercent := 0.0
		if totalTests > 0 {
			reductionPercent = float64(totalTests-selectedCount) / float64(totalTests) * 100
		}

		// Build selection result
		selection := &model.Selection{
			GeneratedAt:         time.Now().UTC(),
			FromCommit:          mapping.CommitSHA,
			ToCommit:            currentCommit,
			ChangedFiles:        changedFiles,
			SelectedTests:       selectedTests,
			OutOfScopeTestFiles: outOfScopeTestFiles,
			Stats: model.SelectionStats{
				TotalTests:          totalTests,
				SelectedTests:       selectedCount,
				ChangedFiles:        len(changedFiles),
				ChangedTestFiles:    changedTestFilesCount,
				OutOfScopeTestFiles: outOfScopeTestFilesCount,
				NewTests:            newTestsCount,
				ReductionPercent:    reductionPercent,
			},
		}

		// Save selection
		if err := jsonutil.SaveSelection(outputPath, selection); err != nil {
			return fmt.Errorf("failed to save selection: %w", err)
		}

		// Print summary
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("Selection Complete!")
		fmt.Printf("  From commit:    %s\n", selection.FromCommit[:12])
		fmt.Printf("  To commit:      %s\n", selection.ToCommit[:12])
		fmt.Printf("  Changed files:  %d\n", len(changedFiles))
		if changedTestFilesCount > 0 {
			fmt.Printf("  Test files:     %d (in scope, all tests in affected packages selected)\n", changedTestFilesCount)
		}
		if outOfScopeTestFilesCount > 0 {
			fmt.Printf("  [Warn] Out-of-scope test files: %d (not in baseline, ignored by RTS)\n", outOfScopeTestFilesCount)
			for _, f := range outOfScopeTestFiles {
				fmt.Printf("         - %s\n", f)
			}
		}
		if newTestsCount > 0 {
			fmt.Printf("  New tests:      %d (not in baseline, selected automatically)\n", newTestsCount)
		}
		if runAll {
			fmt.Printf("  [Warning] RUN-ALL triggered by: %s\n", triggerFile)
		}
		fmt.Printf("  Selected tests: %d/%d (%.1f%% reduction)\n",
			selectedCount, totalTests, reductionPercent)
		fmt.Printf("  Output:         %s\n", outputPath)
		fmt.Println(strings.Repeat("=", 50))

		return nil
	},
}

// getAllChangedFiles returns all changed files (not just .go) for run-all pattern matching
func getAllChangedFiles(repoPath, fromCommit, toCommit, stripPrefix string) ([]string, error) {
	fromCommit = strings.TrimSpace(fromCommit)
	toCommit = strings.TrimSpace(toCommit)

	stdout, stderr, err := exec.Run(repoPath, "git", "diff", "--name-only", fromCommit+".."+toCommit)
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %s", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			if stripPrefix != "" {
				line = strings.TrimPrefix(line, stripPrefix)
			}
			files = append(files, line)
		}
	}
	return files, nil
}

func findTestDirectory(baseline *model.BaselineManifest, testName string) string {
	for _, suite := range baseline.TestSuiteResults {
		for _, result := range suite.TestResults {
			if result.TestName == testName {
				return result.Directory
			}
		}
	}
	return "" // Unknown directory
}

// discoverNewTests finds tests that exist in the repo but are not in the baseline mapping
func discoverNewTests(repoPath string, mapping *model.CoverageMapping, baseline *model.BaselineManifest) ([]string, error) {
	var newTests []string

	// Get unique directories from baseline
	directories := make(map[string]bool)
	for _, suite := range baseline.TestSuiteResults {
		directories[suite.Directory] = true
	}

	// Run go test -list for each directory to discover current tests
	for dir := range directories {
		testDir := dir
		if !strings.HasPrefix(testDir, "./") {
			testDir = "./" + testDir
		}

		stdout, _, err := exec.Run(repoPath, "go", "test", "-list", ".*", testDir)
		if err != nil {
			continue
		}

		// Parse output to get test names
		currentTests := parseGoTestList(stdout, dir)

		// Find tests not in mapping
		for _, qualifiedName := range currentTests {
			if _, exists := mapping.TestToFiles[qualifiedName]; !exists {
				newTests = append(newTests, qualifiedName)
			}
		}
	}

	return newTests, nil
}

// parseGoTestList parses the output of `go test -list` and returns qualified test names
func parseGoTestList(output, directory string) []string {
	var tests []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// go test -list outputs test names, then "ok <package> <time>" or "? <package> [no test files]"
		if strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "? ") {
			continue
		}
		// Valid test names start with "Test", "Benchmark", "Example", or "Fuzz"
		if strings.HasPrefix(line, "Test") || strings.HasPrefix(line, "Benchmark") ||
			strings.HasPrefix(line, "Example") || strings.HasPrefix(line, "Fuzz") {
			qualifiedName := model.QualifyTestName(directory, line)
			tests = append(tests, qualifiedName)
		}
	}
	return tests
}

// findChangedFunctions compares current function checksums against baseline checksums
// to find functions that have changed
func findChangedFunctions(repoPath string, changedFiles []string, baselineChecksums map[string]string) []string {
	var changedFunctions []string

	// Compute current checksums for changed files
	currentChecksums, err := coverage.ComputeAllChecksums(repoPath, changedFiles)
	if err != nil {
		return changedFunctions
	}

	// Compare with baseline checksums
	for qualifiedFunc, currentHash := range currentChecksums {
		baselineHash, exists := baselineChecksums[qualifiedFunc]
		if !exists {
			// New function - consider it changed
			changedFunctions = append(changedFunctions, qualifiedFunc)
		} else if currentHash != baselineHash {
			// Function checksum changed
			changedFunctions = append(changedFunctions, qualifiedFunc)
		}
	}

	// Also check for deleted functions (in baseline but not in current)
	for qualifiedFunc := range baselineChecksums {
		// Extract file from qualified name (file.go::FuncName)
		parts := strings.Split(qualifiedFunc, "::")
		if len(parts) < 2 {
			continue
		}
		file := parts[0]

		// Only check functions from changed files
		for _, changedFile := range changedFiles {
			if file == changedFile {
				if _, exists := currentChecksums[qualifiedFunc]; !exists {
					// Function was deleted - select tests that covered it
					changedFunctions = append(changedFunctions, qualifiedFunc)
				}
				break
			}
		}
	}

	return changedFunctions
}

func init() {
	rootCmd.AddCommand(selectCmd)
	selectCmd.Flags().String("baseline", ".cov/baseline.json", "Path to baseline.json")
	selectCmd.Flags().String("mapping", ".cov/mapping.json", "Path to mapping between test and application code files")
	selectCmd.Flags().String("output", ".cov/selection.json", "Output path for tests selection structure")
	selectCmd.Flags().String("repo", "", "Path to tested git repository from where the tests are executed")
	selectCmd.Flags().String("strip-prefix", "", "Prefix to strip from git diff paths (e.g., ray-operator/)")
	selectCmd.Flags().String("granularity", "file", "Selection granularity: 'file' or 'function'")
	selectCmd.Flags().StringSlice("run-all-on", []string{}, "Patterns that trigger full test run (e.g., go.mod,go.sum,Makefile)")
	selectCmd.MarkFlagRequired("baseline")
	selectCmd.MarkFlagRequired("mapping")
	selectCmd.MarkFlagRequired("repo")
	selectCmd.MarkFlagRequired("strip-prefix")
}
