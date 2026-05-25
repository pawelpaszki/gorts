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
	testRepoPath   string // Path to cloned gorts-demo repository
	gortsBinary    string // Path to built gorts binary
	testBinaryPath string // Path to instrumented test binary
	baselineFile   string // Path to pre-generated baseline.json
	mappingFile    string // Path to pre-generated mapping.json
	coverageDir    string // Path to coverage data directory
)

// fixed temp directory (local and GitHub execution compatible)
const tempDir = "/tmp/gorts-integration"

// Two commits for testing: baseline is generated at the older commit,
// then the newer commit is used, so select tests can detect real changes.
const (
	// baselineCommit: baseline and mapping are generated at this commit
	baselineCommit = "b923c3c60a2e1c3ee1dd396f0be9f4174d62a57f"
	// currentCommit: tests run at this commit (one ahead of baseline)
	currentCommit = "579a4e856c9aa4752637ffad15b4b7a296927f22"
)

func TestMain(m *testing.M) {
	// clean up repo (if exists)
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	// clone gorts-demo and checkout baseline commit first
	// currentCommit will be checked out after generating baseline/mapping
	// this repo has known history and is easier to use than mocking every change
	testRepoPath = filepath.Join(tempDir, "gorts-demo")
	if err := gitClone("https://github.com/pawelpaszki/gorts-demo.git", testRepoPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clone gorts-demo: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}
	if err := configureGitIdentity(testRepoPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure git identity: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}
	if err := gitCheckout(testRepoPath, baselineCommit); err != nil {
		fmt.Fprintf(os.Stderr, "failed to checkout baseline commit %s: %v\n", baselineCommit, err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}
	fmt.Printf("checked out baseline commit: %s for integration tests\n", baselineCommit)

	// build the gorts binary from source and place it in the temp directory
	gortsRoot := findGortsRoot()
	gortsBinary = filepath.Join(tempDir, "gorts")
	if err := goBuild(gortsRoot, gortsBinary); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build gorts: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// build test binary from gorts-demo (shared across tests)
	testBinaryPath = filepath.Join(tempDir, "test.bin")
	if err := buildTestBinary(testRepoPath, testBinaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// setup output paths (inside gorts-demo repo)
	covDir := filepath.Join(testRepoPath, ".cov")
	os.MkdirAll(covDir, 0755)
	coverageDir = filepath.Join(covDir, "coverage")
	baselineFile = filepath.Join(covDir, "baseline.json")
	mappingFile = filepath.Join(covDir, "mapping.json")

	// generate baseline (runs tests, collects coverage)
	_, thisFile, _, _ := runtime.Caller(0)
	manifestFile := filepath.Join(filepath.Dir(thisFile), "testdata", "manifest.json")
	if err := runGortsSetup(testRepoPath, gortsBinary,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", testBinaryPath,
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
	); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate baseline: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// generate mapping (parses coverage data)
	if err := runGortsSetup(testRepoPath, gortsBinary,
		"mapping",
		"--baseline", baselineFile,
		"--output", mappingFile,
		"--module", "github.com/pawelpaszki/gorts-demo",
		"--repo", testRepoPath,
	); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate mapping: %v\n", err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// checkout current commit (one ahead of baseline) for tests to run
	// this allows select tests to detect real changes between commits
	if err := gitCheckout(testRepoPath, currentCommit); err != nil {
		fmt.Fprintf(os.Stderr, "failed to checkout current commit %s: %v\n", currentCommit, err)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}
	fmt.Printf("checked out current commit: %s\n", currentCommit)

	fmt.Printf("test artifacts available here: %s\n", tempDir)
	fmt.Printf("  baseline: %s (generated at %s)\n", baselineFile, baselineCommit[:12])
	fmt.Printf("  mapping: %s (generated at %s)\n", mappingFile, baselineCommit[:12])
	fmt.Printf("  repo now at: %s\n", currentCommit[:12])

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
	cmd := exec.Command("git", "clone", url, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// configureGitIdentity sets a local git author for tests that create commits.
// CI runners often have no global user.name/user.email configured.
func configureGitIdentity(repoPath string) error {
	if err := gitConfigLocal(repoPath, "user.email", "gorts-test@example.com"); err != nil {
		return err
	}
	return gitConfigLocal(repoPath, "user.name", "GoRTS Test")
}

func gitConfigLocal(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// gitCheckout checks out a specific commit in the repository
func gitCheckout(repoPath, commitSHA string) error {
	cmd := exec.Command("git", "checkout", commitSHA)
	cmd.Dir = repoPath
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

// buildTestBinary builds an instrumented test binary from gorts-demo
func buildTestBinary(repoPath, outputPath string) error {
	cmd := exec.Command("go", "test", "-c", "-cover", "-coverpkg=./...", "-o", outputPath, "./test/e2e")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// runGortsSetup runs a gorts command during TestMain setup (no *testing.T available)
func runGortsSetup(workDir, binary string, args ...string) error {
	cmd := exec.Command(binary, args...)
	cmd.Dir = workDir
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
	dir := filepath.Join(testRepoPath, ".cov")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	return dir
}

// fixtureFile returns the absolute path to a fixture file in testdata/
func fixtureFile(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}

// checkoutCommit checks out a specific commit (for use in tests)
func checkoutCommit(t *testing.T, repoPath, commitSHA string) error {
	t.Helper()
	return gitCheckout(repoPath, commitSHA)
}

// gitAdd stages a file for commit
func gitAdd(repoPath, file string) error {
	cmd := exec.Command("git", "add", file)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// gitCommit creates a commit with the given message
func gitCommit(repoPath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// gitResetHard resets the repository to a specific commit, discarding changes
func gitResetHard(repoPath, commitSHA string) error {
	cmd := exec.Command("git", "reset", "--hard", commitSHA)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, output)
	}
	return nil
}

// runGortsInDir executes gorts from a specific working directory with specified parameters
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
