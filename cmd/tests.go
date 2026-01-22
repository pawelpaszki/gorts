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

var testsCmd = &cobra.Command{
	Use:   "tests",
	Short: "Enumerate and persist test names from provided directories",
	Long:  `List out all test names starting with 'Test' from comma-separated directories specified`,
	RunE: func(cmd *cobra.Command, args []string) error {
		directoriesInput, _ := cmd.Flags().GetString("directories")
		if strings.TrimSpace(directoriesInput) == "" {
			return fmt.Errorf("--directories parameter is required!")
		}
		outputDir, _ := cmd.Flags().GetString("output")
		if strings.TrimSpace(outputDir) == "" {
			return fmt.Errorf("--output parameter is required!")
		}
		directories := splitCsv(directoriesInput)
		suiteMap := make(map[string][]string)

		for _, directory := range directories {
			stdout, stderr, err := exec.Run(directory, "go", "test", "-list", "Test.*", "./...")
			if err != nil {
				return fmt.Errorf("failed to list tests in %s: %w\nStderr: %s", directory, err, stderr)
			}

			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			if len(lines) == 1 && strings.Contains(lines[0], "no test files") {
				fmt.Printf("[Warn] No test files found in %s\n", directory)
				continue
			}

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Test") {
					suiteMap[directory] = append(suiteMap[directory], line)
				}
			}
			fmt.Printf("[Info] Found %d tests in %s\n", len(suiteMap[directory]), directory)
		}

		var testSuites []model.TestSuite
		for dir, tests := range suiteMap {
			testSuites = append(testSuites, model.TestSuite{
				Directory: dir,
				Tests:     tests,
			})
		}

		commitSha, _, _ := exec.Run(directories[0], "git", "rev-parse", "HEAD")

		manifest := model.TestManifest{
			GeneratedAt: time.Now().UTC(),
			CommitSHA:   strings.TrimSpace(commitSha),
			TestSuites:  testSuites,
		}
		if err := jsonutil.SaveManifest(outputDir, &manifest); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}
		fmt.Printf("[Info] Saved manifest to %s\n", outputDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testsCmd)
	testsCmd.Flags().String("directories", "", "Comma-separated test suite directories, e.g. `./test/e2e,./test/e2eautoscaler`")
	testsCmd.Flags().String("output", "", "directory to save tests and directories names. It is recommended to save them in the same directory, from which the repo tests are executed")
}

func splitCsv(s string) []string {
	tokens := strings.Split(s, ",")
	values := make([]string, 0, len(tokens))
	for _, v := range tokens {
		v = strings.TrimSpace(v)
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}
