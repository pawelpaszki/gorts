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
	Long: `Run tests and collect per-test coverage data.

Modes (mutually exclusive):
  --test-binary    Use pre-built instrumented binary (for standard Go tests)
  --pre/post-test  Use hooks for coverage collection (for kuberay-style e2e)

Examples:
  # Standard Go tests (build binary first with: go test -c -cover -coverpkg=./... -o test.bin ./test/...)
  gorts baseline --manifest tests.json --test-binary ./test.bin --coverage-dir .cov/coverage --output baseline.json

  # Kuberay-style with hooks
  gorts baseline --manifest tests.json --post-test "kubectl cp pod:/cov {{COVERAGE_PATH}}" --coverage-dir .cov/coverage --output baseline.json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath, _ := cmd.Flags().GetString("manifest")
		outputPath, _ := cmd.Flags().GetString("output")
		coverageDir, _ := cmd.Flags().GetString("coverage-dir")
		testBinary, _ := cmd.Flags().GetString("test-binary")
		preTestCmd, _ := cmd.Flags().GetString("pre-test")
		postTestCmd, _ := cmd.Flags().GetString("post-test")

		// Validate mutual exclusivity (hooks and test-binary)
		if testBinary != "" && (preTestCmd != "" || postTestCmd != "") {
			return fmt.Errorf("--test-binary and --pre-test/--post-test are mutually exclusive\n" +
				"  Use --test-binary for standard Go tests with instrumented binary\n" +
				"  Use --pre-test/--post-test for kuberay-style external coverage collection")
		}

		// Validate test binary exists if specified
		if testBinary != "" {
			if _, err := os.Stat(testBinary); os.IsNotExist(err) {
				return fmt.Errorf("test binary not found: %s\n"+
					"  Build it with: go test -c -cover -coverpkg=./... -o %s ./test/...", testBinary, testBinary)
			}
			if coverageDir == "" {
				return fmt.Errorf("--coverage-dir is required when using --test-binary")
			}
		}

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
		r.MaxRetries, _ = cmd.Flags().GetInt("retry")

		// Configure runner based on mode
		if testBinary != "" {
			// test binary mode
			r.TestBinary = testBinary
			r.CoverageDir = coverageDir
			fmt.Printf("[Info] Using test binary mode: %s\n", testBinary)
		} else {
			// Hook mode
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
					covPath, _ = filepath.Abs(covPath)
					os.MkdirAll(covPath, 0755)
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

		if err := jsonutil.SaveBaseline(outputPath, baseline); err != nil {
			return err
		}

		fmt.Printf("\n==================================================\n")
		fmt.Printf("Baseline Complete!\n")
		fmt.Printf("Tests: %d passed, %d failed, %d total\n",
			overallSummary.Passed, overallSummary.Failed, overallSummary.Total)
		fmt.Printf("Duration: %dms\n", overallSummary.DurationMs)
		fmt.Printf("Output: %s\n", outputPath)
		if coverageDir != "" {
			fmt.Printf("Coverage: %s\n", coverageDir)
		}
		fmt.Printf("==================================================\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(baselineCmd)

	// Required flags
	baselineCmd.Flags().String("manifest", "", "Path to test manifest from 'gorts tests' command")
	baselineCmd.MarkFlagRequired("manifest")
	baselineCmd.Flags().String("output", "", "Path (directory + filename) to save baseline JSON")
	baselineCmd.MarkFlagRequired("output")

	// Coverage flags
	baselineCmd.Flags().String("coverage-dir", "", "Path (directory + filename) to store per-test coverage data")
	baselineCmd.Flags().String("test-binary", "", "Path to pre-built instrumented test binary (mutually exclusive with hooks)")

	// Hook flags (for kuberay-style)
	baselineCmd.Flags().String("pre-test", "", "Command to run before each test (supports {{DIR}}, {{TEST}})")
	baselineCmd.Flags().String("post-test", "", "Command to run after each test (supports {{DIR}}, {{TEST}}, {{COVERAGE_PATH}})")

	// Optional flags
	baselineCmd.Flags().StringSlice("env", []string{}, "Environment variables: KEY=val,KEY2=val2")
	baselineCmd.Flags().StringSlice("skip", []string{}, "Tests to skip: --skip TestFoo --skip TestBar")
	baselineCmd.Flags().Int("retry", 0, "Max retries per test on failure")
}
