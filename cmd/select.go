package cmd

import (
	"fmt"
	"strings"
	"time"

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

		fmt.Printf("DEBUG: mapping.CommitSHA=[%s] len=%d\n", mapping.CommitSHA, len(mapping.CommitSHA))
		fmt.Printf("DEBUG: currentCommit=[%s] len=%d\n", currentCommit, len(currentCommit))

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

		// Get changed files between baseline commit and current
		changedFiles, err := getChangedFiles(repoPath, mapping.CommitSHA, currentCommit, stripPrefix)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		if len(changedFiles) == 0 {
			fmt.Println("No source files changed (only non-code files modified)")
			return nil
		}

		// Select tests based on changed files
		selectedTestsMap := make(map[string]string) // testName -> directory
		for _, file := range changedFiles {
			if tests, ok := mapping.FileToTests[file]; ok {
				for _, testName := range tests {
					// Find directory from baseline
					dir := findTestDirectory(baseline, testName)
					selectedTestsMap[testName] = dir
				}
			}
		}

		// Build selected tests slice
		var selectedTests []model.SelectedTest
		for testName, dir := range selectedTestsMap {
			selectedTests = append(selectedTests, model.SelectedTest{
				Directory: dir,
				TestName:  testName,
			})
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
			GeneratedAt:   time.Now().UTC(),
			FromCommit:    mapping.CommitSHA,
			ToCommit:      currentCommit,
			ChangedFiles:  changedFiles,
			SelectedTests: selectedTests,
			Stats: model.SelectionStats{
				TotalTests:       totalTests,
				SelectedTests:    selectedCount,
				ChangedFiles:     len(changedFiles),
				ReductionPercent: reductionPercent,
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
		fmt.Printf("  Selected tests: %d/%d (%.1f%% reduction)\n",
			selectedCount, totalTests, reductionPercent)
		fmt.Printf("  Output:         %s\n", outputPath)
		fmt.Println(strings.Repeat("=", 50))

		return nil
	},
}

func getChangedFiles(repoPath, fromCommit, toCommit, stripPrefix string) ([]string, error) {
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
		if line != "" && strings.HasSuffix(line, ".go") {
			// Strip prefix if provided
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

func init() {
	rootCmd.AddCommand(selectCmd)
	selectCmd.Flags().String("baseline", ".cov/baseline.json", "Path to baseline.json")
	selectCmd.Flags().String("mapping", ".cov/mapping.json", "Path to mapping between test and application code files")
	selectCmd.Flags().String("output", ".cov/selection.json", "Output path for tests selection structure")
	selectCmd.Flags().String("repo", "", "Path to tested git repository from where the tests are executed")
	selectCmd.Flags().String("strip-prefix", "", "Prefix to strip from git diff paths (e.g., ray-operator/)")
	selectCmd.MarkFlagRequired("baseline")
	selectCmd.MarkFlagRequired("mapping")
	selectCmd.MarkFlagRequired("repo")
	selectCmd.MarkFlagRequired("strip-prefix")

}
