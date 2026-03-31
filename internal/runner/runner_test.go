package runner

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	r := New()

	assert.NotNil(t, r)
	assert.Nil(t, r.PreHook)
	assert.Nil(t, r.PostHook)
	assert.Empty(t, r.Env)
	assert.Equal(t, 0, r.MaxRetries)
	assert.Empty(t, r.TestBinary)
	assert.Empty(t, r.CoverageDir)
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		maxLen int
		want   string
	}{
		{
			name:   "output shorter than max",
			output: "short output",
			maxLen: 100,
			want:   "short output",
		},
		{
			name:   "output equal to max",
			output: "exactly ten",
			maxLen: 11,
			want:   "exactly ten",
		},
		{
			name:   "output longer than max",
			output: "this is a very long output that needs to be truncated",
			maxLen: 20,
			want:   "this is a very long \n... [truncated]",
		},
		{
			name:   "empty output",
			output: "",
			maxLen: 100,
			want:   "",
		},
		{
			name:   "max length zero",
			output: "some output",
			maxLen: 0,
			want:   "\n... [truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOutput(tt.output, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildCoveragePath(t *testing.T) {
	tests := []struct {
		name        string
		coverageDir string
		directory   string
		testName    string
		wantEmpty   bool
		wantContain []string
	}{
		{
			name:        "empty coverage dir returns empty",
			coverageDir: "",
			directory:   "test/integration",
			testName:    "TestAuthor_Create",
			wantEmpty:   true,
		},
		{
			name:        "simple directory",
			coverageDir: "/tmp/coverage",
			directory:   "test/integration",
			testName:    "TestAuthor_Create",
			wantContain: []string{"test_integration", "TestAuthor_Create"},
		},
		{
			name:        "nested directory",
			coverageDir: "/tmp/coverage",
			directory:   "internal/handler/book",
			testName:    "TestBookHandler_Get",
			wantContain: []string{"handler_book", "TestBookHandler_Get"},
		},
		{
			name:        "root directory",
			coverageDir: "/tmp/coverage",
			directory:   ".",
			testName:    "TestMain",
			wantContain: []string{"TestMain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{CoverageDir: tt.coverageDir}

			got := r.buildCoveragePath(tt.directory, tt.testName)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				for _, substr := range tt.wantContain {
					assert.Contains(t, got, substr)
				}
			}
		})
	}
}

func TestBuildTestBinaryCommand(t *testing.T) {
	tests := []struct {
		name         string
		testBinary   string
		directory    string
		testName     string
		coveragePath string
		wantArgs     []string
	}{
		{
			name:         "standard test binary command",
			testBinary:   "/path/to/integration.test",
			directory:    "test/integration",
			testName:     "TestAuthor_Create",
			coveragePath: "/tmp/coverage/TestAuthor_Create",
			wantArgs:     []string{"-test.v", "-test.timeout", "30m", "-test.run", "^TestAuthor_Create$", "-test.gocoverdir", "/tmp/coverage/TestAuthor_Create"},
		},
		{
			name:         "test with special characters in name",
			testBinary:   "./test.bin",
			directory:    "test/e2e",
			testName:     "TestE2E_Book_Create",
			coveragePath: "/tmp/cov",
			wantArgs:     []string{"-test.run", "^TestE2E_Book_Create$"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{TestBinary: tt.testBinary}

			cmd := r.buildTestBinaryCommand(tt.directory, tt.testName, tt.coveragePath)

			assert.Equal(t, tt.testBinary, cmd.Path)
			assert.Equal(t, tt.directory, cmd.Dir)
			for _, arg := range tt.wantArgs {
				assert.Contains(t, cmd.Args, arg)
			}
		})
	}
}

func TestBuildGoTestCommand(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		testName  string
		wantArgs  []string
	}{
		{
			name:      "standard go test command",
			directory: "test/integration",
			testName:  "TestAuthor_Create",
			wantArgs:  []string{"go", "test", "-v", "-run", "^TestAuthor_Create$", "./..."},
		},
		{
			name:      "test with underscores",
			directory: "internal/model",
			testName:  "TestBook_Validate_ISBN",
			wantArgs:  []string{"-run", "^TestBook_Validate_ISBN$"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{}

			cmd := r.buildGoTestCommand(tt.directory, tt.testName)

			assert.Equal(t, tt.directory, cmd.Dir)
			for _, arg := range tt.wantArgs {
				assert.Contains(t, cmd.Args, arg)
			}
			assert.Contains(t, cmd.Args, "-timeout")
			assert.Contains(t, cmd.Args, "-count")
			assert.Contains(t, cmd.Args, "-parallel")
		})
	}
}

func TestBuildCommand(t *testing.T) {
	t.Run("uses test binary when configured", func(t *testing.T) {
		r := &Runner{
			TestBinary: "/path/to/test.bin",
		}

		cmd := r.buildCommand("test/integration", "TestFoo", "/tmp/cov")

		assert.True(t, strings.HasSuffix(cmd.Path, "test.bin") || cmd.Path == "/path/to/test.bin")
	})

	t.Run("uses go test when no binary configured", func(t *testing.T) {
		r := &Runner{}

		cmd := r.buildCommand("test/integration", "TestFoo", "")

		assert.Contains(t, cmd.Args[0], "go")
	})

	t.Run("includes custom environment variables", func(t *testing.T) {
		r := &Runner{
			Env: []string{"CUSTOM_VAR=value", "ANOTHER=test"},
		}

		cmd := r.buildCommand("test/integration", "TestFoo", "")

		assert.Contains(t, cmd.Env, "CUSTOM_VAR=value")
		assert.Contains(t, cmd.Env, "ANOTHER=test")
	})
}

func TestRunPreHook(t *testing.T) {
	t.Run("no hook configured does not panic", func(t *testing.T) {
		r := &Runner{}

		assert.NotPanics(t, func() {
			r.runPreHook("test/integration", "TestFoo")
		})
	})

	t.Run("hook is called with correct arguments", func(t *testing.T) {
		var calledDir, calledTest string

		r := &Runner{
			PreHook: func(dir, test string) error {
				calledDir = dir
				calledTest = test
				return nil
			},
		}

		r.runPreHook("test/integration", "TestAuthor_Create")

		assert.Equal(t, "test/integration", calledDir)
		assert.Equal(t, "TestAuthor_Create", calledTest)
	})

	t.Run("hook error does not panic", func(t *testing.T) {
		r := &Runner{
			PreHook: func(dir, test string) error {
				return assert.AnError
			},
		}

		assert.NotPanics(t, func() {
			r.runPreHook("test/integration", "TestFoo")
		})
	})
}

func TestRunPostHook(t *testing.T) {
	t.Run("no hook configured does not panic", func(t *testing.T) {
		r := &Runner{}
		result := &model.TestResult{}

		assert.NotPanics(t, func() {
			r.runPostHook("test/integration", "TestFoo", result)
		})
	})

	t.Run("hook is called with correct arguments", func(t *testing.T) {
		var calledDir, calledTest string
		var calledResult *model.TestResult

		r := &Runner{
			PostHook: func(dir, test string, result *model.TestResult) error {
				calledDir = dir
				calledTest = test
				calledResult = result
				return nil
			},
		}

		expectedResult := &model.TestResult{TestName: "TestAuthor_Create", Status: "pass"}
		r.runPostHook("test/integration", "TestAuthor_Create", expectedResult)

		assert.Equal(t, "test/integration", calledDir)
		assert.Equal(t, "TestAuthor_Create", calledTest)
		assert.Equal(t, expectedResult, calledResult)
	})

	t.Run("hook can modify result", func(t *testing.T) {
		r := &Runner{
			PostHook: func(dir, test string, result *model.TestResult) error {
				result.CoveragePath = "/custom/path"
				return nil
			},
		}

		result := &model.TestResult{}
		r.runPostHook("test/integration", "TestFoo", result)

		assert.Equal(t, "/custom/path", result.CoveragePath)
	})

	t.Run("hook error does not panic", func(t *testing.T) {
		r := &Runner{
			PostHook: func(dir, test string, result *model.TestResult) error {
				return assert.AnError
			},
		}

		assert.NotPanics(t, func() {
			r.runPostHook("test/integration", "TestFoo", &model.TestResult{})
		})
	})
}

func TestBuildCoveragePath_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	coverageDir := filepath.Join(tmpDir, "coverage")

	r := &Runner{CoverageDir: coverageDir}

	path := r.buildCoveragePath("test/integration", "TestAuthor_Create")

	assert.DirExists(t, path)
	assert.Contains(t, path, "TestAuthor_Create")
}
