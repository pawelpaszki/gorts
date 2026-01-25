package cmd

import (
	"fmt"
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

		manifest, err := jsonutil.LoadManifest(manifestPath)
		if err != nil {
			return err
		}

		envVars, _ := cmd.Flags().GetStringSlice("env")

		r := runner.New()
		r.Env = envVars
		r.PreHook = func(dir, testName string) error {
			// TODO: reset coverage counters
			fmt.Printf("[Pre] Preparing coverage for %s\n", testName)
			return nil
		}
		r.PostHook = func(dir, testName string, result *model.TestResult) error {
			// TODO: flush coverage to file
			fmt.Printf("[Post] Saving coverage for %s\n", testName)
			return nil
		}

		var suiteResults []model.TestSuiteResult

		for _, suite := range manifest.TestSuites {
			var testResults []model.TestResult

			for _, testName := range suite.Tests {
				result, err := r.RunSingleTest(suite.Directory, testName)
				if err != nil {
					return fmt.Errorf("failed to run %s: %w", testName, err)
				}
				testResults = append(testResults, *result)
				fmt.Printf("[Info] %s/%s: %s (%dms)\n", suite.Directory, testName, result.Status, result.DurationMs)
			}

			suiteResults = append(suiteResults, model.TestSuiteResult{
				Directory:   suite.Directory,
				TestResults: testResults,
			})
		}

		commitSha, _, _ := exec.Run(manifest.TestSuites[0].Directory, "git", "rev-parse", "HEAD")

		baseline := &model.BaselineManifest{
			GeneratedAt:      time.Now().UTC(),
			CommitSHA:        strings.TrimSpace(commitSha),
			TestSuiteResults: suiteResults,
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
	baselineCmd.Flags().String("pre-test", "", "Command before each test")
	baselineCmd.Flags().String("post-test", "", "Command after each test")
	baselineCmd.Flags().StringSlice("env", []string{}, "Env vars: KEY=val,KEY2=val2")
}
