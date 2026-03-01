package coverage

import (
	"fmt"
	"strings"

	"github.com/pawelpaszki/gorts/internal/exec"
)

// FunctionCoverage - single function coverage info
type FunctionCoverage struct {
	FilePath     string
	LineNumber   int // starting line number of function
	FunctionName string
	Coverage     float64 // Coverage percentage of the function
}

// ParseFunctionCoverage runs `go tool covdata func` and parses the output
func ParseFunctionCoverage(coverageDir string) ([]FunctionCoverage, error) {
	stdout, stderr, err := exec.Run("", "go", "tool", "covdata", "func", "-i="+coverageDir)
	if err != nil {
		return nil, fmt.Errorf("go tool covdata func failed: %w (stderr: %s)", err, stderr)
	}

	return parseFuncOutput(stdout)
}

func parseFuncOutput(output string) ([]FunctionCoverage, error) {
	var results []FunctionCoverage
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}

		// Format: "filepath:line:\t\tfuncName\t\tcoverage%"
		// Example: "/go/src/github.com/org/repo/pkg/foo.go:25:  Bar  80.0%"

		// Find the colon after line number
		firstColon := strings.Index(line, ":")
		if firstColon == -1 {
			continue
		}
		secondColon := strings.Index(line[firstColon+1:], ":")
		if secondColon == -1 {
			continue
		}
		secondColon += firstColon + 1

		filePath := line[:firstColon]
		lineNumStr := line[firstColon+1 : secondColon]
		rest := strings.TrimSpace(line[secondColon+1:])

		// Parse line number
		var lineNum int
		fmt.Sscanf(lineNumStr, "%d", &lineNum)

		// Split remaining into function name and coverage
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}

		funcName := fields[0]
		coverageStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		var coverage float64
		fmt.Sscanf(coverageStr, "%f", &coverage)

		// Only include functions with non-zero coverage
		if coverage > 0 {
			results = append(results, FunctionCoverage{
				FilePath:     filePath,
				LineNumber:   lineNum,
				FunctionName: funcName,
				Coverage:     coverage,
			})
		}
	}

	return results, nil
}

// QualifyFunction returns a unique identifier for a function
// Format: "relative/path/file.go::FunctionName"
func QualifyFunction(filePath, funcName, modulePath string) string {
	relPath := NormalizeFilePath(filePath, modulePath)
	return relPath + "::" + funcName
}
