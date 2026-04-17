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

type baselineConfig struct {
	manifestPath string
	outputPath   string
	coverageDir  string
	testBinary   string
	preTestCmd   string
	postTestCmd  string
	skipSet      map[string]bool
	envVars      []string
	maxRetries   int
}

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
		cfg := parseBaselineFlags(cmd)

		if err := validateBaselineFlags(cfg); err != nil {
			return err
		}

		manifest, err := jsonutil.LoadManifest(cfg.manifestPath)
		if err != nil {
			return err
		}

		r := configureRunner(cfg)

		startedAt := time.Now().UTC()
		suiteResults, overallSummary, err := runTestSuites(manifest, r, cfg.skipSet)
		if err != nil {
			return err
		}

		baseline := buildBaselineManifest(manifest, suiteResults, overallSummary, startedAt)

		if err := jsonutil.SaveBaseline(cfg.outputPath, baseline); err != nil {
			return err
		}

		printBaselineSummary(overallSummary, baseline.BaselineDurationMs, cfg.outputPath, cfg.coverageDir)
		return nil
	},
}

func parseBaselineFlags(cmd *cobra.Command) baselineConfig {
	skipTests, _ := cmd.Flags().GetStringSlice("skip")
	skipSet := make(map[string]bool)
	for _, t := range skipTests {
		skipSet[t] = true
	}

	envVars, _ := cmd.Flags().GetStringSlice("env")
	maxRetries, _ := cmd.Flags().GetInt("retry")

	manifestPath, _ := cmd.Flags().GetString("manifest")
	outputPath, _ := cmd.Flags().GetString("output")
	coverageDir, _ := cmd.Flags().GetString("coverage-dir")
	testBinary, _ := cmd.Flags().GetString("test-binary")
	preTestCmd, _ := cmd.Flags().GetString("pre-test")
	postTestCmd, _ := cmd.Flags().GetString("post-test")

	return baselineConfig{
		manifestPath: manifestPath,
		outputPath:   outputPath,
		coverageDir:  coverageDir,
		testBinary:   testBinary,
		preTestCmd:   preTestCmd,
		postTestCmd:  postTestCmd,
		skipSet:      skipSet,
		envVars:      envVars,
		maxRetries:   maxRetries,
	}
}

func validateBaselineFlags(cfg baselineConfig) error {
	if cfg.testBinary != "" && (cfg.preTestCmd != "" || cfg.postTestCmd != "") {
		return fmt.Errorf("--test-binary and --pre-test/--post-test are mutually exclusive\n" +
			"  Use --test-binary for standard Go tests with instrumented binary\n" +
			"  Use --pre-test/--post-test for kuberay-style external coverage collection")
	}

	if cfg.testBinary != "" {
		if _, err := os.Stat(cfg.testBinary); os.IsNotExist(err) {
			return fmt.Errorf("test binary not found: %s\n"+
				"  Build it with: go test -c -cover -coverpkg=./... -o %s ./test/...", cfg.testBinary, cfg.testBinary)
		}
	}

	return nil
}

func configureRunner(cfg baselineConfig) *runner.Runner {
	r := runner.New()
	r.Env = cfg.envVars
	r.MaxRetries = cfg.maxRetries

	if cfg.testBinary != "" {
		configureTestBinaryMode(r, cfg)
	} else {
		configureHookMode(r, cfg)
	}

	return r
}

func configureTestBinaryMode(r *runner.Runner, cfg baselineConfig) {
	r.TestBinary = cfg.testBinary
	r.CoverageDir = cfg.coverageDir
	fmt.Printf("[Info] Using test binary mode: %s\n", cfg.testBinary)
}

func configureHookMode(r *runner.Runner, cfg baselineConfig) {
	r.PreHook = createPreHook(cfg.preTestCmd)
	r.PostHook = createPostHook(cfg.postTestCmd, cfg.coverageDir)
}

func createPreHook(preTestCmd string) runner.PreHook {
	return func(dir, testName string) error {
		if preTestCmd == "" {
			return nil
		}
		expanded := expandHookCommand(preTestCmd, dir, testName, "")
		fmt.Printf("[Pre] %s\n", expanded)
		_, stderr, err := exec.Run(dir, "sh", "-c", expanded)
		if err != nil {
			return fmt.Errorf("pre-test failed: %s", stderr)
		}
		return nil
	}
}

func createPostHook(postTestCmd, coverageDir string) runner.PostHook {
	return func(dir, testName string, result *model.TestResult) error {
		if postTestCmd == "" {
			return nil
		}

		covPath := buildCoveragePath(dir, testName, coverageDir)
		if covPath != "" {
			result.CoveragePath = covPath
		}

		expanded := expandHookCommand(postTestCmd, dir, testName, covPath)
		fmt.Printf("[Post] %s\n", expanded)
		_, stderr, err := exec.Run(dir, "sh", "-c", expanded)
		if err != nil {
			fmt.Printf("[Warn] post-test failed: %s\n", stderr)
		}
		return nil
	}
}

func expandHookCommand(cmd, dir, testName, coveragePath string) string {
	expanded := strings.ReplaceAll(cmd, "{{DIR}}", dir)
	expanded = strings.ReplaceAll(expanded, "{{TEST}}", testName)
	expanded = strings.ReplaceAll(expanded, "{{COVERAGE_PATH}}", coveragePath)
	return expanded
}

func buildCoveragePath(dir, testName, coverageDir string) string {
	if coverageDir == "" {
		return ""
	}

	parts := strings.Split(filepath.Clean(dir), string(filepath.Separator))
	var sanitized string
	if len(parts) >= 2 {
		sanitized = parts[len(parts)-2] + "_" + parts[len(parts)-1]
	} else {
		sanitized = parts[len(parts)-1]
	}

	covPath := filepath.Join(coverageDir, sanitized, testName)
	covPath, _ = filepath.Abs(covPath)
	os.MkdirAll(covPath, 0755)
	return covPath
}

func runTestSuites(manifest *model.TestManifest, r *runner.Runner, skipSet map[string]bool) ([]model.TestSuiteResult, model.Summary, error) {
	var suiteResults []model.TestSuiteResult
	var overallSummary model.Summary

	for _, suite := range manifest.TestSuites {
		suiteResult, err := runSingleSuite(suite, r, skipSet)
		if err != nil {
			return nil, model.Summary{}, err
		}

		suiteResults = append(suiteResults, suiteResult)
		overallSummary.Total += suiteResult.Summary.Total
		overallSummary.Passed += suiteResult.Summary.Passed
		overallSummary.Failed += suiteResult.Summary.Failed
		overallSummary.DurationMs += suiteResult.Summary.DurationMs
	}

	return suiteResults, overallSummary, nil
}

func runSingleSuite(suite model.TestSuite, r *runner.Runner, skipSet map[string]bool) (model.TestSuiteResult, error) {
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
			return model.TestSuiteResult{}, fmt.Errorf("failed to run %s: %w", testName, err)
		}

		testResults = append(testResults, *result)
		updateSuiteSummary(&suiteSummary, result)

		fmt.Printf("[Info] %s/%s: %s (%dms)\n", suite.Directory, testName, result.Status, result.DurationMs)
	}

	return model.TestSuiteResult{
		Directory:   suite.Directory,
		TestResults: testResults,
		Summary:     suiteSummary,
	}, nil
}

func updateSuiteSummary(summary *model.Summary, result *model.TestResult) {
	summary.Total++
	summary.DurationMs += result.DurationMs
	if result.Status == "pass" {
		summary.Passed++
	} else {
		summary.Failed++
	}
}

func buildBaselineManifest(manifest *model.TestManifest, suiteResults []model.TestSuiteResult, summary model.Summary, startedAt time.Time) *model.BaselineManifest {
	commitSha, _, _ := exec.Run(manifest.TestSuites[0].Directory, "git", "rev-parse", "HEAD")

	finishedAt := time.Now().UTC()
	baselineDurationMs := finishedAt.Sub(startedAt).Milliseconds()

	return &model.BaselineManifest{
		GeneratedAt:        finishedAt,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		BaselineDurationMs: baselineDurationMs,
		CommitSHA:          strings.TrimSpace(commitSha),
		TestSuiteResults:   suiteResults,
		Summary:            summary,
	}
}

func printBaselineSummary(summary model.Summary, baselineDurationMs int64, outputPath, coverageDir string) {
	fmt.Printf("\n==================================================\n")
	fmt.Printf("Baseline Complete!\n")
	fmt.Printf("Tests: %d passed, %d failed, %d total\n",
		summary.Passed, summary.Failed, summary.Total)
	fmt.Printf("Test Duration: %dms\n", summary.DurationMs)
	fmt.Printf("Baseline Duration: %dms\n", baselineDurationMs)
	fmt.Printf("Output: %s\n", outputPath)
	if coverageDir != "" {
		fmt.Printf("Coverage: %s\n", coverageDir)
	}
	fmt.Printf("==================================================\n")
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
	baselineCmd.MarkFlagRequired("coverage-dir")
	baselineCmd.Flags().String("test-binary", "", "Path to pre-built instrumented test binary (mutually exclusive with hooks)")

	// Hook flags (for kuberay-style)
	baselineCmd.Flags().String("pre-test", "", "Command to run before each test (supports {{DIR}}, {{TEST}})")
	baselineCmd.Flags().String("post-test", "", "Command to run after each test (supports {{DIR}}, {{TEST}}, {{COVERAGE_PATH}})")

	// Optional flags
	baselineCmd.Flags().StringSlice("env", []string{}, "Environment variables: KEY=val,KEY2=val2")
	baselineCmd.Flags().StringSlice("skip", []string{}, "Tests to skip: --skip TestFoo --skip TestBar")
	baselineCmd.Flags().Int("retry", 0, "Max retries per test on failure")
}
