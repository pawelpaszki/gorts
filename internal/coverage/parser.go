package coverage

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ParseCoverageFile reads a Go coverage profile and returns covered file names.
// Handles both text format (.out files) and Go 1.20+ binary format (directories with covmeta/covcounters).
// Only returns files with count > 0 (actually executed).
func ParseCoverageFile(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// If it's a directory, assume Go 1.20+ binary format
	if info.IsDir() {
		return parseBinaryCoverage(path)
	}

	// Otherwise, parse as text format
	return parseTextCoverage(path)
}

// parseBinaryCoverage converts Go 1.20+ binary coverage format to text and parses it
func parseBinaryCoverage(dir string) ([]string, error) {
	// Create temp file for the converted output
	tmpFile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Convert binary to text: go tool covdata textfmt -i=<dir> -o=<tmpfile>
	cmd := exec.Command("go", "tool", "covdata", "textfmt", "-i="+dir, "-o="+tmpFile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go tool covdata textfmt: %v: %s", err, string(out))
	}

	// Parse the converted text file
	return parseTextCoverage(tmpFile.Name())
}

// parseTextCoverage parses a text-format coverage profile
func parseTextCoverage(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileSet := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip mode line and empty lines
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}

		// Format: filename:startLine.startCol,endLine.endCol numStatements count
		// Example: github.com/org/repo/pkg/foo.go:10.1,12.2 1 1
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Get count (last field)
		count := parts[len(parts)-1]
		if count == "0" {
			continue // Not covered
		}

		// Extract filename (before the colon with line numbers)
		filePart := parts[0]
		// fmt.Printf("filename: %s\n", filePart)
		colonIdx := strings.LastIndex(filePart, ":")
		if colonIdx == -1 {
			continue
		}
		fileName := filePart[:colonIdx]

		fileSet[fileName] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Convert set to slice
	files := make([]string, 0, len(fileSet))
	for f := range fileSet {
		files = append(files, f)
	}
	return files, nil
}

// NormalizeFilePath converts a coverage profile path to a relative path.
//
// Coverage profiles contain paths like:
//   - "github.com/ray-project/kuberay/ray-operator/controllers/ray/raycluster_controller.go" (standard)
//   - "/go/src/github.com/ray-project/kuberay/ray-operator/controllers/ray/raycluster_controller.go" (container with GOPATH)
//   - "/app/github.com/ray-project/kuberay/ray-operator/controllers/ray/raycluster_controller.go" (container with custom mount)
//   - "/home/user/go/src/github.com/ray-project/kuberay/ray-operator/controllers/ray/raycluster_controller.go" (local GOPATH)
//
// This function finds the module path within the full path and strips everything
// before it (including the module path itself), returning just the relative path
// within the module (e.g., "controllers/ray/raycluster_controller.go").
func NormalizeFilePath(fullPath, modulePath string) string {
	idx := strings.Index(fullPath, modulePath)
	if idx != -1 {
		// Strip everything up to and including the module path
		relativePath := fullPath[idx+len(modulePath):]
		// Remove leading slash if present
		return strings.TrimPrefix(relativePath, "/")
	}
	// Fallback: return as-is if module path not found
	return fullPath
}

// FindCoverageFiles finds coverage data in a directory.
// Returns the directory itself if it contains Go 1.20+ binary format (covmeta.* files),
// otherwise returns all .out files found recursively.
func FindCoverageFiles(dir string) ([]string, error) {
	// Check for Go 1.20+ binary format (covmeta.* files)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "covmeta.") {
			// Binary format detected - return the directory itself
			return []string{dir}, nil
		}
	}

	// Fallback: look for .out files recursively
	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".out") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
