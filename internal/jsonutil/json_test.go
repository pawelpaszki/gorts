package jsonutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadManifest(t *testing.T) {
	t.Run("save and load valid manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "manifest.json")

		original := &model.TestManifest{
			GeneratedAt: time.Now().UTC().Truncate(time.Second),
			CommitSHA:   "abc123def456",
			TestSuites: []model.TestSuite{
				{
					Directory: "test/integration",
					Tests:     []string{"TestAuthor_Create", "TestBook_Create"},
				},
			},
		}

		err := SaveManifest(path, original)
		require.NoError(t, err)

		loaded, err := LoadManifest(path)
		require.NoError(t, err)

		assert.Equal(t, original.CommitSHA, loaded.CommitSHA)
		assert.Equal(t, original.TestSuites, loaded.TestSuites)
	})

	t.Run("save creates parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "deep", "manifest.json")

		manifest := &model.TestManifest{
			GeneratedAt: time.Now(),
			CommitSHA:   "abc123",
			TestSuites:  []model.TestSuite{{Directory: "test"}},
		}

		err := SaveManifest(path, manifest)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})
}

func TestLoadManifest(t *testing.T) {
	t.Run("load non-existent file returns error", func(t *testing.T) {
		_, err := LoadManifest("/nonexistent/path/manifest.json")
		assert.Error(t, err)
	})

	t.Run("load invalid JSON returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(path, []byte("not valid json"), 0644)
		require.NoError(t, err)

		_, err = LoadManifest(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest JSON")
	})

	t.Run("load JSON with unknown fields returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "unknown_field.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "abc123",
			"test_suites": [{"directory": "test", "tests": []}],
			"unknown_field": "should fail"
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadManifest(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest JSON")
	})

	t.Run("load JSON failing validation returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid_manifest.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "",
			"test_suites": [{"directory": "test", "tests": []}]
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadManifest(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest")
	})
}

func TestSaveAndLoadBaseline(t *testing.T) {
	t.Run("save and load valid baseline", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "baseline.json")

		original := &model.BaselineManifest{
			GeneratedAt: time.Now().UTC().Truncate(time.Second),
			CommitSHA:   "abc123def456",
			TestSuiteResults: []model.TestSuiteResult{
				{
					Directory: "test/integration",
					TestResults: []model.TestResult{
						{
							Directory:  "test/integration",
							TestName:   "TestAuthor_Create",
							Status:     "pass",
							DurationMs: 150,
						},
					},
					Summary: model.Summary{Total: 1, Passed: 1},
				},
			},
			Summary: model.Summary{Total: 1, Passed: 1, DurationMs: 150},
		}

		err := SaveBaseline(path, original)
		require.NoError(t, err)

		loaded, err := LoadBaseline(path)
		require.NoError(t, err)

		assert.Equal(t, original.CommitSHA, loaded.CommitSHA)
		assert.Equal(t, original.Summary.Total, loaded.Summary.Total)
		assert.Len(t, loaded.TestSuiteResults, 1)
	})

	t.Run("save creates parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "deep", "baseline.json")

		baseline := &model.BaselineManifest{
			GeneratedAt:      time.Now(),
			CommitSHA:        "abc123",
			TestSuiteResults: []model.TestSuiteResult{{Directory: "test"}},
			Summary:          model.Summary{Total: 1},
		}

		err := SaveBaseline(path, baseline)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})
}

func TestLoadBaseline(t *testing.T) {
	t.Run("load non-existent file returns error", func(t *testing.T) {
		_, err := LoadBaseline("/nonexistent/path/baseline.json")
		assert.Error(t, err)
	})

	t.Run("load invalid JSON returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(path, []byte("not valid json"), 0644)
		require.NoError(t, err)

		_, err = LoadBaseline(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid baseline manifest JSON")
	})

	t.Run("load JSON with unknown fields returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "unknown_field.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "abc123",
			"test_suite_results": [{"directory": "test", "test_results": []}],
			"summary": {"total": 1},
			"unknown_field": "should fail"
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadBaseline(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid baseline manifest JSON")
	})

	t.Run("load JSON failing validation returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid_baseline.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "",
			"test_suite_results": [{"directory": "test"}],
			"summary": {"total": 1}
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadBaseline(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid baseline manifest")
	})
}

func TestSaveAndLoadMapping(t *testing.T) {
	t.Run("save and load valid mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "mapping.json")

		original := &model.CoverageMapping{
			GeneratedAt: time.Now().UTC().Truncate(time.Second),
			CommitSHA:   "abc123def456",
			FileToTests: map[string][]string{
				"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
			},
			TestToFiles: map[string][]string{
				"test/integration/TestBook_Create": {"internal/handler/book_handler.go"},
			},
			Stats: model.MappingStats{
				TotalTests: 1,
				TotalFiles: 1,
			},
		}

		err := SaveMapping(path, original)
		require.NoError(t, err)

		loaded, err := LoadMapping(path)
		require.NoError(t, err)

		assert.Equal(t, original.CommitSHA, loaded.CommitSHA)
		assert.Equal(t, original.Stats.TotalTests, loaded.Stats.TotalTests)
		assert.Len(t, loaded.FileToTests, 1)
	})

	t.Run("save creates parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "deep", "mapping.json")

		mapping := &model.CoverageMapping{
			GeneratedAt: time.Now(),
			CommitSHA:   "abc123",
			FileToTests: map[string][]string{"file.go": {"test"}},
			Stats:       model.MappingStats{TotalTests: 1, TotalFiles: 1},
		}

		err := SaveMapping(path, mapping)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})
}

func TestLoadMapping(t *testing.T) {
	t.Run("load non-existent file returns error", func(t *testing.T) {
		_, err := LoadMapping("/nonexistent/path/mapping.json")
		assert.Error(t, err)
	})

	t.Run("load invalid JSON returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(path, []byte("not valid json"), 0644)
		require.NoError(t, err)

		_, err = LoadMapping(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mapping JSON")
	})

	t.Run("load JSON with unknown fields returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "unknown_field.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "abc123",
			"file_to_tests": {"file.go": ["test"]},
			"test_to_files": {"test": ["file.go"]},
			"stats": {"total_tests": 1, "total_files": 1},
			"unknown_field": "should fail"
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadMapping(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mapping JSON")
	})

	t.Run("load JSON failing validation returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid_mapping.json")
		content := `{
			"generated_at": "2024-01-01T00:00:00Z",
			"commit_sha": "",
			"file_to_tests": {"file.go": ["test"]},
			"stats": {"total_tests": 1, "total_files": 1}
		}`
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadMapping(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid coverage mapping manifest")
	})
}

func TestSaveSelection(t *testing.T) {
	t.Run("save valid selection", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "selection.json")

		selection := &model.Selection{
			GeneratedAt: time.Now().UTC().Truncate(time.Second),
			FromCommit:  "abc123",
			ToCommit:    "def456",
			ChangedFiles: []string{
				"internal/handler/book_handler.go",
			},
			SelectedTests: []model.SelectedTest{
				{Directory: "test/integration", TestName: "TestBook_Create"},
			},
			Stats: model.SelectionStats{
				TotalTests:    10,
				SelectedTests: 3,
			},
		}

		err := SaveSelection(path, selection)
		require.NoError(t, err)
		assert.FileExists(t, path)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "abc123")
		assert.Contains(t, string(content), "def456")
	})

	t.Run("save creates parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "deep", "selection.json")

		selection := &model.Selection{
			GeneratedAt: time.Now(),
			FromCommit:  "abc123",
			ToCommit:    "def456",
		}

		err := SaveSelection(path, selection)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})
}
