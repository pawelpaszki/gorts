package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTaintingFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{
			name: "go source file",
			file: "internal/handler/book_handler.go",
			want: true,
		},
		{
			name: "go test file",
			file: "internal/model/book_test.go",
			want: true,
		},
		{
			name: "go.mod file",
			file: "go.mod",
			want: true,
		},
		{
			name: "go.sum file",
			file: "go.sum",
			want: true,
		},
		{
			name: "nested go file",
			file: "pkg/stringutil/stringutil.go",
			want: true,
		},
		{
			name: "markdown file",
			file: "README.md",
			want: false,
		},
		{
			name: "json file",
			file: "config.json",
			want: false,
		},
		{
			name: "yaml file",
			file: "docker-compose.yaml",
			want: false,
		},
		{
			name: "makefile",
			file: "Makefile",
			want: false,
		},
		{
			name: "dockerfile",
			file: "Dockerfile",
			want: false,
		},
		{
			name: "shell script",
			file: "scripts/test.sh",
			want: false,
		},
		{
			name: "gitignore",
			file: ".gitignore",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTaintingFile(tt.file)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyCleanRepo(t *testing.T) {
	t.Run("clean repo returns no error", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		err := VerifyCleanRepo(repoPath)
		assert.NoError(t, err)
	})

	t.Run("uncommitted go file returns error", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		// Create uncommitted Go file
		goFile := filepath.Join(repoPath, "main.go")
		err := os.WriteFile(goFile, []byte("package main\n"), 0644)
		require.NoError(t, err)

		err = VerifyCleanRepo(repoPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "uncommitted changes")
		assert.Contains(t, err.Error(), "main.go")
	})

	t.Run("uncommitted go.mod returns error", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		// Create uncommitted go.mod
		goMod := filepath.Join(repoPath, "go.mod")
		err := os.WriteFile(goMod, []byte("module test\n"), 0644)
		require.NoError(t, err)

		err = VerifyCleanRepo(repoPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "go.mod")
	})

	t.Run("uncommitted go.sum returns error", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		// Create uncommitted go.sum
		goSum := filepath.Join(repoPath, "go.sum")
		err := os.WriteFile(goSum, []byte("github.com/example v1.0.0\n"), 0644)
		require.NoError(t, err)

		err = VerifyCleanRepo(repoPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "go.sum")
	})

	t.Run("uncommitted non-go file is allowed", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		// Create uncommitted non-Go file
		readmeFile := filepath.Join(repoPath, "README.md")
		err := os.WriteFile(readmeFile, []byte("# Test\n"), 0644)
		require.NoError(t, err)

		err = VerifyCleanRepo(repoPath)
		assert.NoError(t, err)
	})

	t.Run("multiple uncommitted go files returns error with all files", func(t *testing.T) {
		repoPath := setupTestRepo(t)

		// Create multiple uncommitted Go files
		err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte("package main\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(repoPath, "util.go"), []byte("package main\n"), 0644)
		require.NoError(t, err)

		err = VerifyCleanRepo(repoPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "main.go")
		assert.Contains(t, err.Error(), "util.go")
	})
}

func TestVerifyAtCommit(t *testing.T) {
	t.Run("matching commit returns no error", func(t *testing.T) {
		repoPath := setupTestRepo(t)
		commitSHA := getHeadCommit(t, repoPath)

		err := VerifyAtCommit(repoPath, commitSHA)
		assert.NoError(t, err)
	})

	t.Run("different commit returns error", func(t *testing.T) {
		repoPath := setupTestRepo(t)
		currentCommit := getHeadCommit(t, repoPath)

		// Create a new commit
		newFile := filepath.Join(repoPath, "new.txt")
		err := os.WriteFile(newFile, []byte("new content\n"), 0644)
		require.NoError(t, err)
		runGit(t, repoPath, "add", "new.txt")
		runGit(t, repoPath, "commit", "-m", "second commit")

		// Verify against old commit should fail
		err = VerifyAtCommit(repoPath, currentCommit)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "baseline was generated at")
	})

	t.Run("non-existent repo returns error", func(t *testing.T) {
		err := VerifyAtCommit("/nonexistent/path", "abc123")
		assert.Error(t, err)
	})
}

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()

	// Initialize git repo
	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.email", "test@test.com")
	runGit(t, repoPath, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(repoPath, "initial.txt")
	err := os.WriteFile(initialFile, []byte("initial content\n"), 0644)
	require.NoError(t, err)

	runGit(t, repoPath, "add", "initial.txt")
	runGit(t, repoPath, "commit", "-m", "initial commit")

	return repoPath
}

// runGit runs a git command in the specified directory
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(output))
}

// getHeadCommit returns the current HEAD commit SHA
func getHeadCommit(t *testing.T, repoPath string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	require.NoError(t, err)
	return string(output[:len(output)-1]) // trim newline
}
