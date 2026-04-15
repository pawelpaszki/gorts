package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pawelpaszki/gorts/internal/coverage"
	"github.com/pawelpaszki/gorts/internal/exec"
	"github.com/pawelpaszki/gorts/internal/helpers"
	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/spf13/cobra"
)

type selectionResult struct {
	selectedTests          []model.SelectedTest
	outOfScopeTestFiles    []string
	changedTestFiles       int
	newTestsCount          int
	noCoverageDataPackages []string // packages with changed test files but no coverage data
}

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
		runAllPatterns, _ := cmd.Flags().GetStringSlice("run-all-on")

		baseline, mapping, err := loadBaselineAndMapping(baselinePath, mappingPath)
		if err != nil {
			return err
		}

		currentCommit, err := getCurrentCommit(repoPath)
		if err != nil {
			return err
		}

		warnIfCommitMismatch(baseline, mapping)

		if mapping.CommitSHA == currentCommit {
			printNoChangesDetected(mapping.CommitSHA, currentCommit)
			return nil
		}

		allChangedFiles, err := getAllChangedFiles(repoPath, mapping.CommitSHA, currentCommit, stripPrefix)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		if len(allChangedFiles) == 0 {
			fmt.Println("[Info] No files changed between commits")
			selection := buildEmptySelection(mapping.CommitSHA, currentCommit, mapping.Stats.TotalTests, "no_changes")
			if err := jsonutil.SaveSelection(outputPath, selection); err != nil {
				return fmt.Errorf("failed to save selection: %w", err)
			}
			printSelectionSummary(selection, outputPath, false, "")
			return nil
		}

		runAll, triggerFile := helpers.CheckRunAllTrigger(allChangedFiles, runAllPatterns)
		changedGoFiles := filterGoFiles(allChangedFiles)

		if !runAll && len(changedGoFiles) == 0 {
			fmt.Println("[Info] No source files changed (only non-code files modified)")
			selection := buildEmptySelection(mapping.CommitSHA, currentCommit, mapping.Stats.TotalTests, "no_source_changes")
			if err := jsonutil.SaveSelection(outputPath, selection); err != nil {
				return fmt.Errorf("failed to save selection: %w", err)
			}
			printSelectionSummary(selection, outputPath, false, "")
			return nil
		}

		baselineDirs := buildBaselineDirs(baseline)

		var result selectionResult
		if runAll {
			fmt.Printf("[Info] Run-all triggered by: %s\n", triggerFile)
			result.selectedTests = selectAllTests(mapping)
		} else {
			result = performTestSelection(repoPath, changedGoFiles, baselineDirs, mapping, baseline, granularity)
		}

		selection := buildSelection(mapping.CommitSHA, currentCommit, changedGoFiles, result, mapping.Stats.TotalTests)

		if err := jsonutil.SaveSelection(outputPath, selection); err != nil {
			return fmt.Errorf("failed to save selection: %w", err)
		}

		printSelectionSummary(selection, outputPath, runAll, triggerFile)
		return nil
	},
}

func loadBaselineAndMapping(baselinePath, mappingPath string) (*model.BaselineManifest, *model.CoverageMapping, error) {
	baseline, err := jsonutil.LoadBaseline(baselinePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load baseline: %w", err)
	}

	mapping, err := jsonutil.LoadMapping(mappingPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load mapping: %w", err)
	}

	return baseline, mapping, nil
}

func getCurrentCommit(repoPath string) (string, error) {
	stdout, _, err := exec.Run(repoPath, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(stdout), nil
}

func warnIfCommitMismatch(baseline *model.BaselineManifest, mapping *model.CoverageMapping) {
	if baseline.CommitSHA != mapping.CommitSHA {
		fmt.Printf("[Warn] Baseline commit (%s) differs from mapping commit (%s)\n",
			baseline.CommitSHA[:12], mapping.CommitSHA[:12])
	}
}

func printNoChangesDetected(baselineCommit, currentCommit string) {
	fmt.Println("==================================================")
	fmt.Println("No changes detected!")
	fmt.Printf("  Baseline commit: %s\n", baselineCommit)
	fmt.Printf("  Current commit:  %s\n", currentCommit)
	fmt.Println("  Recommendation: No tests need to be run")
	fmt.Println("==================================================")
}

func filterGoFiles(files []string) []string {
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
	}
	return goFiles
}

func buildBaselineDirs(baseline *model.BaselineManifest) map[string]bool {
	dirs := make(map[string]bool)
	for _, suite := range baseline.TestSuiteResults {
		dirs[suite.Directory] = true
	}
	return dirs
}

func selectAllTests(mapping *model.CoverageMapping) []model.SelectedTest {
	var tests []model.SelectedTest
	for qualifiedName := range mapping.TestToFiles {
		dir, testName := helpers.ParseQualifiedTest(qualifiedName)
		tests = append(tests, model.SelectedTest{
			Directory: dir,
			TestName:  testName,
		})
	}
	return tests
}

func performTestSelection(
	repoPath string,
	changedGoFiles []string,
	baselineDirs map[string]bool,
	mapping *model.CoverageMapping,
	baseline *model.BaselineManifest,
	granularity string,
) selectionResult {
	sourceFiles, inScopeTestFiles, outOfScopeTestFiles := categorizeChangedFiles(changedGoFiles, baselineDirs)

	selectedTestsMap := make(map[string]bool)

	if granularity == "function" {
		if len(mapping.FunctionChecksums) > 0 {
			selectByFunction(repoPath, sourceFiles, mapping, selectedTestsMap)
		} else {
			fmt.Println("[Warn] Function-level granularity requested but no checksums in mapping")
			fmt.Println("[Warn] Falling back to file-level (run mapping with --repo to enable function-level)")
			selectByFile(sourceFiles, mapping, selectedTestsMap)
		}
	} else {
		selectByFile(sourceFiles, mapping, selectedTestsMap)
	}

	noCoveragePackages := selectFromChangedTestFiles(inScopeTestFiles, mapping, selectedTestsMap)

	newTestsCount := discoverAndSelectNewTests(repoPath, mapping, baseline, selectedTestsMap)

	return selectionResult{
		selectedTests:          buildSelectedTestsSlice(selectedTestsMap),
		outOfScopeTestFiles:    outOfScopeTestFiles,
		changedTestFiles:       len(inScopeTestFiles),
		newTestsCount:          newTestsCount,
		noCoverageDataPackages: noCoveragePackages,
	}
}

func categorizeChangedFiles(changedFiles []string, baselineDirs map[string]bool) (sourceFiles, inScopeTestFiles, outOfScopeTestFiles []string) {
	for _, file := range changedFiles {
		if strings.HasSuffix(file, "_test.go") {
			pkgDir := filepath.Dir(file)
			if isInBaselineDirs(pkgDir, baselineDirs) {
				inScopeTestFiles = append(inScopeTestFiles, file)
			} else {
				outOfScopeTestFiles = append(outOfScopeTestFiles, file)
			}
		} else {
			sourceFiles = append(sourceFiles, file)
		}
	}
	return
}

// isInBaselineDirs checks if pkgDir matches any baseline directory using suffix matching
// This handles cases where baseline dirs use relative paths (../../repo/test/e2e)
// but git diff outputs repo-relative paths (test/e2e)
func isInBaselineDirs(pkgDir string, baselineDirs map[string]bool) bool {
	// Exact match first
	if baselineDirs[pkgDir] {
		return true
	}
	// Suffix match: check if any baseline dir ends with the changed file's directory
	// e.g., baseline "../../repo/test/e2erayjob" ends with "/test/e2erayjob" matches "test/e2erayjob"
	for dir := range baselineDirs {
		if strings.HasSuffix(dir, "/"+pkgDir) || strings.HasSuffix(dir, string(filepath.Separator)+pkgDir) {
			return true
		}
	}
	return false
}

func selectByFunction(repoPath string, sourceFiles []string, mapping *model.CoverageMapping, selectedTestsMap map[string]bool) {
	changedFunctions := findChangedFunctions(repoPath, sourceFiles, mapping.FunctionChecksums)
	fmt.Printf("[Info] Function-level analysis: %d changed functions detected\n", len(changedFunctions))

	for _, qualifiedFunc := range changedFunctions {
		if tests, ok := mapping.FunctionToTests[qualifiedFunc]; ok {
			for _, qualifiedName := range tests {
				selectedTestsMap[qualifiedName] = true
			}
		}
	}
}

func selectByFile(sourceFiles []string, mapping *model.CoverageMapping, selectedTestsMap map[string]bool) {
	for _, file := range sourceFiles {
		if tests, ok := mapping.FileToTests[file]; ok {
			for _, qualifiedName := range tests {
				selectedTestsMap[qualifiedName] = true
			}
		}
	}
}

func selectFromChangedTestFiles(inScopeTestFiles []string, mapping *model.CoverageMapping, selectedTestsMap map[string]bool) []string {
	var noCoveragePackages []string
	seenPackages := make(map[string]bool)

	for _, testFile := range inScopeTestFiles {
		pkgDir := filepath.Dir(testFile)
		foundMatch := false

		for qualifiedName := range mapping.TestToFiles {
			dir, _ := helpers.ParseQualifiedTest(qualifiedName)
			if dirMatchesSuffix(dir, pkgDir) {
				selectedTestsMap[qualifiedName] = true
				foundMatch = true
			}
		}

		if !foundMatch && !seenPackages[pkgDir] {
			noCoveragePackages = append(noCoveragePackages, pkgDir)
			seenPackages[pkgDir] = true
		}
	}

	return noCoveragePackages
}

// dirMatchesSuffix checks if two directory paths match, accounting for different path formats
// e.g., "../../repo/test/e2erayjob" should match "test/e2erayjob"
func dirMatchesSuffix(baselineDir, changedDir string) bool {
	if baselineDir == changedDir {
		return true
	}
	// Check if baseline dir ends with the changed dir
	// This handles cases where baseline uses relative paths from working directory
	if strings.HasSuffix(baselineDir, "/"+changedDir) || strings.HasSuffix(baselineDir, string(filepath.Separator)+changedDir) {
		return true
	}
	return false
}

func discoverAndSelectNewTests(repoPath string, mapping *model.CoverageMapping, baseline *model.BaselineManifest, selectedTestsMap map[string]bool) int {
	newTests, err := discoverNewTests(repoPath, mapping, baseline)
	if err != nil {
		fmt.Printf("[Warn] Failed to discover new tests: %v\n", err)
		return 0
	}

	for _, qualifiedName := range newTests {
		selectedTestsMap[qualifiedName] = true
	}

	if len(newTests) > 0 {
		fmt.Printf("[Info] Discovered %d new test(s) not in baseline\n", len(newTests))
	}

	return len(newTests)
}

func buildSelectedTestsSlice(selectedTestsMap map[string]bool) []model.SelectedTest {
	tests := make([]model.SelectedTest, 0, len(selectedTestsMap))
	for qualifiedName := range selectedTestsMap {
		dir, testName := helpers.ParseQualifiedTest(qualifiedName)
		tests = append(tests, model.SelectedTest{
			Directory: dir,
			TestName:  testName,
		})
	}
	return tests
}

func buildEmptySelection(fromCommit, toCommit string, totalTests int, reason string) *model.Selection {
	return &model.Selection{
		GeneratedAt:         time.Now().UTC(),
		FromCommit:          fromCommit,
		ToCommit:            toCommit,
		ChangedFiles:        []string{},
		SelectedTests:       []model.SelectedTest{},
		OutOfScopeTestFiles: []string{},
		Stats: model.SelectionStats{
			TotalTests:          totalTests,
			SelectedTests:       0,
			ChangedFiles:        0,
			ChangedTestFiles:    0,
			OutOfScopeTestFiles: 0,
			NewTests:            0,
			ReductionPercent:    100.0,
		},
	}
}

func buildSelection(fromCommit, toCommit string, changedFiles []string, result selectionResult, totalTests int) *model.Selection {
	selectedCount := len(result.selectedTests)
	reductionPercent := calculateReductionPercent(totalTests, selectedCount)

	return &model.Selection{
		GeneratedAt:            time.Now().UTC(),
		FromCommit:             fromCommit,
		ToCommit:               toCommit,
		ChangedFiles:           changedFiles,
		SelectedTests:          result.selectedTests,
		OutOfScopeTestFiles:    result.outOfScopeTestFiles,
		NoCoverageDataPackages: result.noCoverageDataPackages,
		Stats: model.SelectionStats{
			TotalTests:          totalTests,
			SelectedTests:       selectedCount,
			ChangedFiles:        len(changedFiles),
			ChangedTestFiles:    result.changedTestFiles,
			OutOfScopeTestFiles: len(result.outOfScopeTestFiles),
			NewTests:            result.newTestsCount,
			ReductionPercent:    reductionPercent,
		},
	}
}

func calculateReductionPercent(total, selected int) float64 {
	if total <= 0 {
		return 0.0
	}
	reduction := float64(total-selected) / float64(total) * 100
	if reduction < 0 {
		return 0.0
	}
	return reduction
}

func printSelectionSummary(selection *model.Selection, outputPath string, runAll bool, triggerFile string) {
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Selection Complete!")
	fmt.Printf("  From commit:    %s\n", selection.FromCommit[:12])
	fmt.Printf("  To commit:      %s\n", selection.ToCommit[:12])
	fmt.Printf("  Changed files:  %d\n", selection.Stats.ChangedFiles)

	if selection.Stats.ChangedTestFiles > 0 {
		fmt.Printf("  Test files:     %d (in scope, all tests in affected packages selected)\n", selection.Stats.ChangedTestFiles)
	}
	if len(selection.NoCoverageDataPackages) > 0 {
		fmt.Printf("  [Warn] %d package(s) with no coverage data (tests were likely skipped during baseline):\n", len(selection.NoCoverageDataPackages))
		for _, pkg := range selection.NoCoverageDataPackages {
			fmt.Printf("         - %s\n", pkg)
		}
	}
	if selection.Stats.OutOfScopeTestFiles > 0 {
		fmt.Printf("  [Warn] Out-of-scope test files: %d (not in baseline, ignored by RTS)\n", selection.Stats.OutOfScopeTestFiles)
		for _, f := range selection.OutOfScopeTestFiles {
			fmt.Printf("         - %s\n", f)
		}
	}
	if selection.Stats.NewTests > 0 {
		fmt.Printf("  New tests:      %d (not in baseline, selected automatically)\n", selection.Stats.NewTests)
	}
	if runAll {
		fmt.Printf("  [Warning] RUN-ALL triggered by: %s\n", triggerFile)
	}

	fmt.Printf("  Selected tests: %d/%d (%.1f%% reduction)\n",
		selection.Stats.SelectedTests, selection.Stats.TotalTests, selection.Stats.ReductionPercent)
	fmt.Printf("  Output:         %s\n", outputPath)
	fmt.Println(strings.Repeat("=", 50))
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
			qualifiedName := helpers.QualifyTestName(directory, line)
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
