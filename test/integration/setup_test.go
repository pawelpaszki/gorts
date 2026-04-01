//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// shared variables (across this and all other integration test files)
var (
	testRepoPath string // Path to cloned gorts-demo repository
	gortsBinary  string // Path to built gorts binary
)

// fixed temp directory (local and GitHub execution compatible)
const tempDir = "/tmp/gorts-test"

func TestMain(m *testing.M) {
	// clean up repo (if exists)
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	// clone gorts-demo to have controlled repo with known commit history
	testRepoPath = filepath.Join(tempDir, "gorts-demo")
	if err := gitClone("https://github.com/pawelpaszki/gorts-demo.git", testRepoPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clone gorts-demo: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// build the gorts binary from source and place it in the temp directory
	gortsRoot := findGortsRoot()
	gortsBinary = filepath.Join(tempDir, "gorts")
	if err := goBuild(gortsRoot, gortsBinary); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build gorts: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	fmt.Printf("test artifacts available here: %s\n", tempDir)

	// run all integration tests
	code := m.Run()

	// Cleanup (skip if GORTS_KEEP_ARTIFACTS is set for debugging)
	if os.Getenv("GORTS_KEEP_ARTIFACTS") == "" {
		os.RemoveAll(tempDir)
	} else {
		fmt.Printf("keeping artifacts in: %s\n", tempDir)
	}
	os.Exit(code)
}

// gitClone clones desired directory into the specified location
func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// goBuild builds the gorts binary
func goBuild(srcDir, outputPath string) error {
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = srcDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// findGortsRoot returns the path to the gorts repository root
// it navigates up from this test file's location
func findGortsRoot() string {
	// this file is at: gorts/test/integration/setup_test.go
	// so gorts root is: ../../
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// outputDir returns the shared output directory path, creating it if needed
func outputDir(t *testing.T) string {
	t.Helper()
	dir := "/tmp/gorts-test/.cov"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	return dir
}

// runGortsInDir executes gorts from a specific working directory with specifid parameters
func runGortsInDir(t *testing.T, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(gortsBinary, args...)
	cmd.Dir = workDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()

	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout, stderr, exitCode
}
