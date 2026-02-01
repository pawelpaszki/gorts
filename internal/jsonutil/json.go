package jsonutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pawelpaszki/gorts/internal/model"
)

// SaveManifest saves test manifest, including commitSha, generated_at (timestamp)
// and test suites consisting of list of directories and tests found in each
// of the respective directories
// used in the Phase 1: get all tests (gorts 'tests')
func SaveManifest(path string, manifest *model.TestManifest) error {
	createDirectoryIfNotPresent(path)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	saveFile(path, data)
	fmt.Printf("[Info] Saved test manifest to %s\n", path)
	return nil
}

// LoadManifest loads test manifest obtained previously (using gorts 'tests')
// the manifest is subsequently used to run the test suites from directories
// specified in the loaded json data - any missing data results in an error
func LoadManifest(path string) (*model.TestManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var manifest model.TestManifest
	decoder := json.NewDecoder(f)
	// next line throws an error if unknown field is present in the json data
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if err := model.ValidateTestManifest(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	return &manifest, nil
}

func SaveBaseline(path string, baseline *model.BaselineManifest) error {
	// create directory if does not exist, then saves (or overwrites) the .json file
	createDirectoryIfNotPresent(path)
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}
	saveFile(path, data)
	fmt.Printf("[Info] Saved baseline to %s\n", path)
	return nil
}

func LoadBaseline(path string) (*model.BaselineManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var baselineManifest model.BaselineManifest
	decoder := json.NewDecoder(f)
	// next line throws an error if unknown field is present in the json data
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&baselineManifest); err != nil {
		return nil, fmt.Errorf("invalid baseline manifest JSON: %w", err)
	}
	if err := model.ValidateBaselineManifest(&baselineManifest); err != nil {
		return nil, fmt.Errorf("invalid baseline manifest: %w", err)
	}
	return &baselineManifest, nil
}

func createDirectoryIfNotPresent(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func saveFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return nil
}
