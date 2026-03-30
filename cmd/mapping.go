package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pawelpaszki/gorts/internal/coverage"
	"github.com/pawelpaszki/gorts/internal/gitutil"
	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/spf13/cobra"
)

type fileLevelResult struct {
	fileToTests    map[string][]string
	testToFiles    map[string][]string
	testsProcessed int
	testsSkipped   int
}

type functionLevelResult struct {
	functionToTests   map[string][]string
	testToFunctions   map[string][]string
	allCoveredFiles   []string
	functionChecksums map[string]string
}

var mappingCmd = &cobra.Command{
	Use:   "mapping",
	Short: "Build mapping from baseline coverage data",
	Long:  "Build test-to-file and file-to-test mappings from baseline coverage data",
	RunE: func(cmd *cobra.Command, args []string) error {
		baselinePath, _ := cmd.Flags().GetString("baseline")
		outputPath, _ := cmd.Flags().GetString("output")
		modulePath, _ := cmd.Flags().GetString("module")
		repoPath, _ := cmd.Flags().GetString("repo")

		baseline, err := jsonutil.LoadBaseline(baselinePath)
		if err != nil {
			return fmt.Errorf("loading baseline: %w", err)
		}

		if err := verifyRepoState(repoPath, baseline.CommitSHA); err != nil {
			return err
		}

		fileResult := buildFileLevelMapping(baseline, modulePath)
		stats := calculateStats(fileResult.fileToTests, fileResult.testToFiles)

		funcResult := buildFunctionLevelMapping(baseline, modulePath, repoPath)
		updateFunctionStats(&stats, funcResult)

		mapping := &model.CoverageMapping{
			GeneratedAt:       time.Now().UTC(),
			CommitSHA:         baseline.CommitSHA,
			FileToTests:       fileResult.fileToTests,
			TestToFiles:       fileResult.testToFiles,
			FunctionToTests:   funcResult.functionToTests,
			TestToFunctions:   funcResult.testToFunctions,
			FunctionChecksums: funcResult.functionChecksums,
			Stats:             stats,
		}

		if err := jsonutil.SaveMapping(outputPath, mapping); err != nil {
			return fmt.Errorf("saving mapping: %w", err)
		}

		printMappingSummary(fileResult, stats, outputPath)
		return nil
	},
}

func verifyRepoState(repoPath, commitSHA string) error {
	if repoPath == "" {
		return nil
	}

	if err := gitutil.VerifyAtCommit(repoPath, commitSHA); err != nil {
		return fmt.Errorf("commit mismatch: %w", err)
	}

	if err := gitutil.VerifyCleanRepo(repoPath); err != nil {
		return fmt.Errorf("repo not clean: %w", err)
	}

	return nil
}

func buildFileLevelMapping(baseline *model.BaselineManifest, modulePath string) fileLevelResult {
	fileToTests := make(map[string][]string)
	testToFiles := make(map[string][]string)
	testsProcessed := 0
	testsSkipped := 0

	for _, suite := range baseline.TestSuiteResults {
		for _, result := range suite.TestResults {
			if result.CoveragePath == "" {
				testsSkipped++
				continue
			}

			coveredFiles, ok := parseCoverageForTest(result, modulePath)
			if !ok {
				testsSkipped++
				continue
			}

			qualifiedName := model.QualifyTestName(result.Directory, result.TestName)
			testToFiles[qualifiedName] = coveredFiles

			for _, file := range coveredFiles {
				fileToTests[file] = appendUnique(fileToTests[file], qualifiedName)
			}

			testsProcessed++
			fmt.Printf("[Info] %s covers %d files\n", qualifiedName, len(coveredFiles))
		}
	}

	return fileLevelResult{
		fileToTests:    fileToTests,
		testToFiles:    testToFiles,
		testsProcessed: testsProcessed,
		testsSkipped:   testsSkipped,
	}
}

func parseCoverageForTest(result model.TestResult, modulePath string) ([]string, bool) {
	coverageFiles, err := findCoverageInPath(result.CoveragePath)
	if err != nil || len(coverageFiles) == 0 {
		fmt.Printf("[Warn] No coverage files for %s in %s\n", result.TestName, result.CoveragePath)
		return nil, false
	}

	var allCoveredFiles []string
	for _, covFile := range coverageFiles {
		files, err := coverage.ParseCoverageFile(covFile)
		if err != nil {
			fmt.Printf("[Warn] Failed to parse %s: %v\n", covFile, err)
			continue
		}
		allCoveredFiles = append(allCoveredFiles, files...)
	}

	uniqueFiles := deduplicate(allCoveredFiles)
	if modulePath != "" {
		for i, f := range uniqueFiles {
			uniqueFiles[i] = coverage.NormalizeFilePath(f, modulePath)
		}
	}

	return uniqueFiles, true
}

func buildFunctionLevelMapping(baseline *model.BaselineManifest, modulePath, repoPath string) functionLevelResult {
	functionToTests := make(map[string][]string)
	testToFunctions := make(map[string][]string)
	var allCoveredFiles []string

	for _, suite := range baseline.TestSuiteResults {
		for _, result := range suite.TestResults {
			if result.CoveragePath == "" {
				continue
			}

			qualifiedTestName := model.QualifyTestName(result.Directory, result.TestName)
			funcCoverage, err := coverage.ParseFunctionCoverage(result.CoveragePath)
			if err != nil {
				fmt.Printf("[Warn] Failed to parse function coverage for %s: %v\n", result.TestName, err)
				continue
			}

			for _, fc := range funcCoverage {
				qualifiedFunc := coverage.QualifyFunction(fc.FilePath, fc.FunctionName, modulePath)
				functionToTests[qualifiedFunc] = appendUnique(functionToTests[qualifiedFunc], qualifiedTestName)
				testToFunctions[qualifiedTestName] = appendUnique(testToFunctions[qualifiedTestName], qualifiedFunc)

				relFile := coverage.NormalizeFilePath(fc.FilePath, modulePath)
				allCoveredFiles = appendUnique(allCoveredFiles, relFile)
			}
		}
	}

	functionChecksums := computeFunctionChecksums(repoPath, allCoveredFiles)

	return functionLevelResult{
		functionToTests:   functionToTests,
		testToFunctions:   testToFunctions,
		allCoveredFiles:   allCoveredFiles,
		functionChecksums: functionChecksums,
	}
}

func computeFunctionChecksums(repoPath string, allCoveredFiles []string) map[string]string {
	if repoPath == "" {
		return nil
	}

	checksums, err := coverage.ComputeAllChecksums(repoPath, allCoveredFiles)
	if err != nil {
		fmt.Printf("[Warn] Failed to compute function checksums: %v\n", err)
		return nil
	}

	fmt.Printf("[Info] Computed checksums for %d functions\n", len(checksums))
	return checksums
}

func updateFunctionStats(stats *model.MappingStats, funcResult functionLevelResult) {
	stats.TotalFunctions = len(funcResult.functionToTests)

	if stats.TotalTests > 0 {
		totalFuncsPerTest := 0
		for _, funcs := range funcResult.testToFunctions {
			totalFuncsPerTest += len(funcs)
		}
		stats.AvgFunctionsPerTest = float64(totalFuncsPerTest) / float64(stats.TotalTests)
	}
}

func printMappingSummary(result fileLevelResult, stats model.MappingStats, outputPath string) {
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Mapping Complete!\n")
	fmt.Printf("  Tests processed: %d\n", result.testsProcessed)
	fmt.Printf("  Tests skipped:   %d\n", result.testsSkipped)
	fmt.Printf("  Total files:     %d\n", stats.TotalFiles)
	fmt.Printf("  Avg files/test:  %.1f\n", stats.AvgFilesPerTest)
	fmt.Printf("  Avg tests/file:  %.1f\n", stats.AvgTestsPerFile)
	fmt.Printf("  Output:          %s\n", outputPath)
	fmt.Println(strings.Repeat("=", 50))
}

// findCoverageInPath looks for coverage files in the given path
func findCoverageInPath(path string) ([]string, error) {
	// Check if it's a directory with covcounters files (Go 1.20+ format)
	files, err := coverage.FindCoverageFiles(path)
	if err == nil && len(files) > 0 {
		return files, nil
	}

	// Check if path itself is a .out file
	if strings.HasSuffix(path, ".out") {
		return []string{path}, nil
	}

	// Look for .out files in the directory
	pattern := filepath.Join(path, "*.out")
	matches, _ := filepath.Glob(pattern)
	return matches, nil
}

func calculateStats(fileToTests, testToFiles map[string][]string) model.MappingStats {
	totalTests := len(testToFiles)
	totalFiles := len(fileToTests)

	var totalFilesPerTest, totalTestsPerFile int
	for _, files := range testToFiles {
		totalFilesPerTest += len(files)
	}
	for _, tests := range fileToTests {
		totalTestsPerFile += len(tests)
	}

	var avgFilesPerTest, avgTestsPerFile float64
	if totalTests > 0 {
		avgFilesPerTest = float64(totalFilesPerTest) / float64(totalTests)
	}
	if totalFiles > 0 {
		avgTestsPerFile = float64(totalTestsPerFile) / float64(totalFiles)
	}

	return model.MappingStats{
		TotalTests:      totalTests,
		TotalFiles:      totalFiles,
		AvgFilesPerTest: avgFilesPerTest,
		AvgTestsPerFile: avgTestsPerFile,
	}
}

func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func init() {
	rootCmd.AddCommand(mappingCmd)
	mappingCmd.Flags().String("baseline", ".cov/baseline.json", "Path to baseline.json")
	mappingCmd.Flags().String("output", ".cov/mapping.json", "Output path for mapping")
	mappingCmd.Flags().String("module", "", "parameter normalizes coverage file paths to relative paths, enabling accurate correlation between instrumented code coverage data and source control change detection")
	mappingCmd.Flags().String("repo", "", "Path to git repository (required for function-level checksums)")
	mappingCmd.MarkFlagRequired("module")
}
