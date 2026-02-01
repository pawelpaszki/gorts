package cmd

import (
	"fmt"

	"github.com/pawelpaszki/gorts/internal/jsonutil"
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

		// Load baseline
		baseline, err := jsonutil.LoadBaseline(baselinePath)
		if err != nil {
			return fmt.Errorf("loading baseline: %w", err)
		}

		fmt.Printf("[Info] Loaded baseline with %d test suites\n", len(baseline.TestSuiteResults))

		fmt.Printf("[Info] Saving mapping to: %s\n", outputPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mappingCmd)
	mappingCmd.Flags().String("baseline", ".cov/baseline.json", "Path to baseline.json")
	mappingCmd.Flags().String("module", "", "Go module path to strip from file names (optional)")
	mappingCmd.MarkFlagRequired("mapping")
}
