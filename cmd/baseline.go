package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pawelpaszki/gorts/internal/exec"
	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/pawelpaszki/gorts/internal/runner"
	"github.com/spf13/cobra"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Run tests and collect coverage",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath, _ := cmd.Flags().GetString("manifest")
		outputPath, _ := cmd.Flags().GetString("output")

		skipTests, _ := cmd.Flags().GetStringSlice("skip")
		skipSet := make(map[string]bool)
		for _, t := range skipTests {
			skipSet[t] = true
		}

		manifest, err := jsonutil.LoadManifest(manifestPath)
		if err != nil {
			return err
		}

		envVars, _ := cmd.Flags().GetStringSlice("env")

		r := runner.New()
		r.Env = envVars
		preTestCmd, _ := cmd.Flags().GetString("pre-test")
		postTestCmd, _ := cmd.Flags().GetString("post-test")
		coverageDir, _ := cmd.Flags().GetString("coverage-dir")

		r.PreHook = func(dir, testName string) error {
			if preTestCmd == "" {
				return nil
			}
			expanded := strings.ReplaceAll(preTestCmd, "{{DIR}}", dir)
			expanded = strings.ReplaceAll(expanded, "{{TEST}}", testName)
			fmt.Printf("[Pre] %s\n", expanded)
			_, stderr, err := exec.Run(dir, "sh", "-c", expanded)
			if err != nil {
				return fmt.Errorf("pre-test failed: %s", stderr)
			}
			return nil
		}

		r.PostHook = func(dir, testName string, result *model.TestResult) error {
			if postTestCmd == "" {
				return nil
			}
			// Compute coverage path (directory for coverage data) - use absolute path
			covPath := ""
			if coverageDir != "" {
				parts := strings.Split(filepath.Clean(dir), string(filepath.Separator))
				var sanitized string
				if len(parts) >= 2 {
					sanitized = parts[len(parts)-2] + "_" + parts[len(parts)-1]
				} else {
					sanitized = parts[len(parts)-1]
				}
				covPath = filepath.Join(coverageDir, sanitized, testName)
				covPath, _ = filepath.Abs(covPath) // Make absolute for shell commands
				os.MkdirAll(covPath, 0755)         // Create the target directory itself
				result.CoveragePath = covPath
			}

			expanded := strings.ReplaceAll(postTestCmd, "{{DIR}}", dir)
			expanded = strings.ReplaceAll(expanded, "{{TEST}}", testName)
			expanded = strings.ReplaceAll(expanded, "{{COVERAGE_PATH}}", covPath)
			fmt.Printf("[Post] %s\n", expanded)
			_, stderr, err := exec.Run(dir, "sh", "-c", expanded)
			if err != nil {
				fmt.Printf("[Warn] post-test failed: %s\n", stderr)
			}
			return nil
		}

		var suiteResults []model.TestSuiteResult
		var overallSummary model.Summary

		for _, suite := range manifest.TestSuites {
			var testResults []model.TestResult
			var suiteSummary model.Summary

			for _, testName := range suite.Tests {
				if skipSet[testName] {
					fmt.Printf("[Skip] %s\n", testName)
					continue
				}
				fmt.Printf("[Info] Running: %s/%s\n", suite.Directory, testName)
				result, err := r.RunSingleTest(suite.Directory, testName)
				if err != nil {
					return fmt.Errorf("failed to run %s: %w", testName, err)
				}
				testResults = append(testResults, *result)

				// Update suite summary
				suiteSummary.Total++
				suiteSummary.DurationMs += result.DurationMs
				if result.Status == "pass" {
					suiteSummary.Passed++
				} else {
					suiteSummary.Failed++
				}

				fmt.Printf("[Info] %s/%s: %s (%dms)\n", suite.Directory, testName, result.Status, result.DurationMs)
			}

			suiteResults = append(suiteResults, model.TestSuiteResult{
				Directory:   suite.Directory,
				TestResults: testResults,
				Summary:     suiteSummary,
			})

			// Accumulate overall summary
			overallSummary.Total += suiteSummary.Total
			overallSummary.Passed += suiteSummary.Passed
			overallSummary.Failed += suiteSummary.Failed
			overallSummary.DurationMs += suiteSummary.DurationMs
		}

		commitSha, _, _ := exec.Run(manifest.TestSuites[0].Directory, "git", "rev-parse", "HEAD")

		baseline := &model.BaselineManifest{
			GeneratedAt:      time.Now().UTC(),
			CommitSHA:        strings.TrimSpace(commitSha),
			TestSuiteResults: suiteResults,
			Summary:          overallSummary,
		}

		return jsonutil.SaveBaseline(outputPath, baseline)
	},
}

func init() {
	rootCmd.AddCommand(baselineCmd)
	baselineCmd.Flags().String("manifest", "", "Path to existing test manifest obtained using gorts 'tests' command, e.g. somedir/tests.json")
	baselineCmd.MarkFlagRequired("manifest")
	baselineCmd.Flags().String("output", "", "Path (directory + filename) to save baseline output")
	baselineCmd.MarkFlagRequired("output")
	baselineCmd.Flags().String("coverage-dir", "", "Path (directory + filename) to save baseline output")
	baselineCmd.Flags().String("pre-test", "", "Command before each test")
	baselineCmd.Flags().String("post-test", "", "Command after each test")
	baselineCmd.Flags().StringSlice("env", []string{}, "Env vars: KEY=val,KEY2=val2")
	baselineCmd.Flags().StringSlice("skip", []string{}, "Tests to skip, e.g. --skip TestFoo --skip TestBar")
}
