package cmd

import (
	"fmt"
	"time"

	"github.com/pawelpaszki/gorts/internal/jsonutil"
	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/spf13/cobra"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Run tests and collect coverage",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath, _ := cmd.Flags().GetString("manifest")

		// Load manifest previously obtained test manifest or return an error (e.g. if does not exist)
		manifest, err := jsonutil.LoadManifest(manifestPath)
		if err != nil {
			return err
		}

		// Loop and print test names and directories for now
		for _, suite := range manifest.TestSuites {
			for _, testName := range suite.Tests {
				fmt.Printf("[TODO] Would run: %s in %s\n", testName, suite.Directory)
			}
		}
		// save empty baseline for now
		outputPath, err := cmd.Flags().GetString("output")
		baseline := &model.BaselineManifest{GeneratedAt: time.Now()}
		jsonutil.SaveBaseline(outputPath, baseline)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(baselineCmd)
	baselineCmd.Flags().String("manifest", "", "Path to existing test manifest obtained using gorts 'tests' command, e.g. somedir/tests.json")
	baselineCmd.MarkFlagRequired("manifest")
	baselineCmd.Flags().String("output", "", "Path (directory + filename) to save baseline output")
	baselineCmd.MarkFlagRequired("output")
}
