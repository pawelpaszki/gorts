package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pawelpaszki/gorts/internal/coverage"
	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/spf13/cobra"
)

/* details about how go coverage works
https://pkg.go.dev/golang.org/x/tools/cover
*/

var mappingCmd = &cobra.Command{
	Use:   "mapping",
	Short: "Build mapping from baseline coverage data",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		baselinePath, _ := cmd.Flags().GetString("baseline")
		outputPath, _ := cmd.Flags().GetString("output")
		modulePath, _ := cmd.Flags().GetString("module")

		// Load baseline
		baseline, err := jsonutil.LoadBaseline(baselinePath)
		if err != nil {
			return fmt.Errorf("loading baseline: %w", err)
		}

		// Initialize maps
		fileToTests := make(map[string][]string)
		testToFiles := make(map[string][]string)

		testsProcessed := 0
		testsSkipped := 0

		// Process each test result
		for _, suite := range baseline.TestSuiteResults {
			for _, result := range suite.TestResults {
				// Skip if no coverage path or test failed
				if result.CoveragePath == "" {
					testsSkipped++
					continue
				}

				// Find coverage files in the coverage path directory
				coverageFiles, err := findCoverageInPath(result.CoveragePath)
				if err != nil || len(coverageFiles) == 0 {
					fmt.Printf("[Warn] No coverage files for %s in %s\n", result.TestName, result.CoveragePath)
					testsSkipped++
					continue
				}

				// Parse all coverage files for this test
				var allCoveredFiles []string
				for _, covFile := range coverageFiles {
					files, err := coverage.ParseCoverageFile(covFile)
					if err != nil {
						fmt.Printf("[Warn] Failed to parse %s: %v\n", covFile, err)
						continue
					}
					allCoveredFiles = append(allCoveredFiles, files...)
				}

				// Deduplicate and normalize
				uniqueFiles := deduplicate(allCoveredFiles)
				if modulePath != "" {
					for i, f := range uniqueFiles {
						// this is required, so that the files appear as if they were
						// shown from the root directory, from which the tests are executed
						uniqueFiles[i] = coverage.NormalizeFilePath(f, modulePath)
					}
				}

				// Build mappings with fq names
				qualifiedName := model.QualifyTestName(result.Directory, result.TestName)
				testToFiles[qualifiedName] = uniqueFiles
				for _, file := range uniqueFiles {
					fileToTests[file] = appendUnique(fileToTests[file], qualifiedName)
				}

				testsProcessed++
				fmt.Printf("[Info] %s covers %d files\n", qualifiedName, len(uniqueFiles))
			}
		}

		// Calculate stats
		stats := calculateStats(fileToTests, testToFiles)

		// Build mapping struct
		mapping := &model.CoverageMapping{
			GeneratedAt: time.Now().UTC(),
			CommitSHA:   baseline.CommitSHA,
			FileToTests: fileToTests,
			TestToFiles: testToFiles,
			Stats:       stats,
		}

		// Save
		if err := jsonutil.SaveMapping(outputPath, mapping); err != nil {
			return fmt.Errorf("saving mapping: %w", err)
		}

		// Print summary
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Mapping Complete!\n")
		fmt.Printf("  Tests processed: %d\n", testsProcessed)
		fmt.Printf("  Tests skipped:   %d\n", testsSkipped)
		fmt.Printf("  Total files:     %d\n", stats.TotalFiles)
		fmt.Printf("  Avg files/test:  %.1f\n", stats.AvgFilesPerTest)
		fmt.Printf("  Avg tests/file:  %.1f\n", stats.AvgTestsPerFile)
		fmt.Printf("  Output:          %s\n", outputPath)
		fmt.Println(strings.Repeat("=", 50))

		return nil
	},
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
	mappingCmd.MarkFlagRequired("module")
}
