//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBaselineCmd_TestBinaryMode(t *testing.T) {
	t.Log("loading manifest fixture")
	manifestFile := fixtureFile(t, "manifest.json")

	t.Log("building instrumented test binary from gorts-demo")
	testBinary := filepath.Join(tempDir, "test.bin")
	buildCmd := exec.Command("go", "test", "-c", "-cover", "-coverpkg=./...", "-o", testBinary, "./test/e2e")
	buildCmd.Dir = testRepoPath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, output)
	}

	coverageDir := filepath.Join(outputDir(t), "coverage")
	baselineFile := filepath.Join(outputDir(t), "baseline.json")

	t.Log("running gorts baseline")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", testBinary,
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
	)
	if exitCode != 0 {
		t.Fatalf("gorts baseline failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("baseline created at %s", baselineFile)

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Fatalf("expected baseline file %s to exist", baselineFile)
	}

	data, err := os.ReadFile(baselineFile)
	if err != nil {
		t.Fatalf("failed to read baseline file: %v", err)
	}

	var baseline struct {
		GeneratedAt      string `json:"generated_at"`
		CommitSHA        string `json:"commit_sha"`
		TestSuiteResults []struct {
			Directory   string `json:"directory"`
			TestResults []struct {
				TestName     string `json:"test_name"`
				Status       string `json:"status"`
				CoveragePath string `json:"coverage_path"`
			} `json:"test_results"`
		} `json:"test_suite_results"`
		Summary struct {
			Total  int `json:"total"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatalf("failed to parse baseline JSON: %v", err)
	}

	t.Log("verifying baseline structure")

	if baseline.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if baseline.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}
	if len(baseline.TestSuiteResults) == 0 {
		t.Error("expected at least one test suite result")
	}
	if baseline.Summary.Total == 0 {
		t.Error("expected summary.total > 0")
	}

	t.Log("verifying coverage was collected")

	if len(baseline.TestSuiteResults) > 0 && len(baseline.TestSuiteResults[0].TestResults) > 0 {
		firstTest := baseline.TestSuiteResults[0].TestResults[0]
		if firstTest.CoveragePath == "" {
			t.Error("expected coverage_path to be set for test results")
		}
		if _, err := os.Stat(firstTest.CoveragePath); os.IsNotExist(err) {
			t.Errorf("coverage path %s does not exist", firstTest.CoveragePath)
		}
	}

	t.Logf("baseline complete: %d tests (%d passed, %d failed)",
		baseline.Summary.Total, baseline.Summary.Passed, baseline.Summary.Failed)
}

func TestBaselineCmd_SkipTests(t *testing.T) {
	t.Log("loading manifest fixture")
	manifestFile := fixtureFile(t, "manifest.json")

	t.Log("building instrumented test binary from gorts-demo")
	testBinary := filepath.Join(tempDir, "test.bin")
	buildCmd := exec.Command("go", "test", "-c", "-cover", "-coverpkg=./...", "-o", testBinary, "./test/e2e")
	buildCmd.Dir = testRepoPath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, output)
	}

	coverageDir := filepath.Join(outputDir(t), "coverage")
	baselineFile := filepath.Join(outputDir(t), "baseline.json")

	t.Log("running gorts baseline with --skip TestE2E_RootEndpoint")
	stdout, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", testBinary,
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
		"--skip", "TestE2E_RootEndpoint",
	)
	if exitCode != 0 {
		t.Fatalf("gorts baseline failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	t.Logf("baseline created at %s", baselineFile)

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Fatalf("expected baseline file %s to exist", baselineFile)
	}

	data, err := os.ReadFile(baselineFile)
	if err != nil {
		t.Fatalf("failed to read baseline file: %v", err)
	}

	var baseline struct {
		GeneratedAt      string `json:"generated_at"`
		CommitSHA        string `json:"commit_sha"`
		TestSuiteResults []struct {
			Directory   string `json:"directory"`
			TestResults []struct {
				TestName     string `json:"test_name"`
				Status       string `json:"status"`
				CoveragePath string `json:"coverage_path"`
			} `json:"test_results"`
		} `json:"test_suite_results"`
		Summary struct {
			Total  int `json:"total"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatalf("failed to parse baseline JSON: %v", err)
	}

	t.Log("verifying baseline structure")

	if baseline.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	if baseline.CommitSHA == "" {
		t.Error("commit_sha should not be empty")
	}
	if len(baseline.TestSuiteResults) == 0 {
		t.Error("expected at least one test suite result")
	}
	if baseline.Summary.Total != 2 {
		t.Error("expected summary.total == 2 (3 tests minus 1 skipped)")
	}

	t.Log("verifying coverage was collected")

	if len(baseline.TestSuiteResults) > 0 && len(baseline.TestSuiteResults[0].TestResults) > 0 {
		firstTest := baseline.TestSuiteResults[0].TestResults[0]
		if firstTest.CoveragePath == "" {
			t.Error("expected coverage_path to be set for test results")
		}
		if _, err := os.Stat(firstTest.CoveragePath); os.IsNotExist(err) {
			t.Errorf("coverage path %s does not exist", firstTest.CoveragePath)
		}
	}

	t.Logf("baseline complete: %d tests (%d passed, %d failed)",
		baseline.Summary.Total, baseline.Summary.Passed, baseline.Summary.Failed)
}

func TestBaselineCmd_MutuallyExclusiveFlags(t *testing.T) {
	t.Log("running gorts baseline with mutually exclusive flags (expecting failure)")

	manifestFile := fixtureFile(t, "manifest.json")
	baselineFile := filepath.Join(outputDir(t), "baseline.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", "/some/binary",
		"--pre-test", "echo pre",
		"--output", baselineFile,
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for mutually exclusive flags")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for mutually exclusive flags")
}

func TestBaselineCmd_MissingTestBinary(t *testing.T) {
	t.Log("running gorts baseline with nonexistent test binary (expecting failure)")

	manifestFile := fixtureFile(t, "manifest.json")
	coverageDir := filepath.Join(outputDir(t), "coverage")
	baselineFile := filepath.Join(outputDir(t), "baseline.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", manifestFile,
		"--test-binary", "/nonexistent/test.bin",
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing test binary")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for missing test binary")
}

func TestBaselineCmd_MissingManifest(t *testing.T) {
	t.Log("running gorts baseline with nonexistent manifest (expecting failure)")

	coverageDir := filepath.Join(outputDir(t), "coverage")
	baselineFile := filepath.Join(outputDir(t), "baseline.json")

	_, stderr, exitCode := runGortsInDir(t, testRepoPath,
		"baseline",
		"--manifest", "/nonexistent/manifest.json",
		"--test-binary", "/some/binary",
		"--coverage-dir", coverageDir,
		"--output", baselineFile,
	)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing manifest")
	}
	if stderr == "" {
		t.Error("expected error message in stderr")
	}

	t.Log("got expected error for missing manifest")
}
