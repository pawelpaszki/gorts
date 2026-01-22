package jsonutil

import (
	"encoding/json"
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
