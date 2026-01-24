package jsonutil

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pawelpaszki/gorts/internal/model"
)

func SaveManifest(path string, manifest *model.TestManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

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
