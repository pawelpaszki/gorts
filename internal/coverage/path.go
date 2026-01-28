package coverage

import (
	"os"
	"path/filepath"
	"strings"
)

func CoveragePath(baseDir, directory, testName string) string {
	sanitized := strings.ReplaceAll(strings.TrimPrefix(directory, "./"), "/", "_")
	path := filepath.Join(baseDir, sanitized, testName+".cov")
	os.MkdirAll(filepath.Dir(path), 0755) // create directory if does not exist
	return path
}
