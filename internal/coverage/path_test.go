package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoveragePath(t *testing.T) {
	tests := []struct {
		name         string
		directory    string
		testName     string
		wantContains []string
	}{
		{
			name:         "generate path for standard test",
			directory:    "internal/model",
			testName:     "TestBook_Validate",
			wantContains: []string{"internal_model", "TestBook_Validate.cov"},
		},
		{
			name:         "generate path for e2e test",
			directory:    "test/e2e",
			testName:     "TestE2E_CreateBook",
			wantContains: []string{"test_e2e", "TestE2E_CreateBook.cov"},
		},
		{
			name:         "generate path for integration test",
			directory:    "test/integration",
			testName:     "TestIntegration_AuthorCRUD",
			wantContains: []string{"test_integration", "TestIntegration_AuthorCRUD.cov"},
		},
		{
			name:         "generate path for handler test",
			directory:    "internal/handler",
			testName:     "TestBookHandler_Get",
			wantContains: []string{"internal_handler", "TestBookHandler_Get.cov"},
		},
		{
			name:         "generate path for service test",
			directory:    "internal/service",
			testName:     "TestAuthorService_Create",
			wantContains: []string{"internal_service", "TestAuthorService_Create.cov"},
		},
		{
			name:         "handle empty directory",
			directory:    "",
			testName:     "TestMain",
			wantContains: []string{"TestMain.cov"},
		},
		{
			name:         "handle nested directory path",
			directory:    "internal/repository/postgres",
			testName:     "TestBookRepo_FindByISBN",
			wantContains: []string{"internal_repository_postgres", "TestBookRepo_FindByISBN.cov"},
		},
		{
			name:         "strip ./ prefix from directory",
			directory:    "./pkg/stringutil",
			testName:     "TestTruncate",
			wantContains: []string{"pkg_stringutil", "TestTruncate.cov"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()

			path := CoveragePath(baseDir, tt.directory, tt.testName)

			for _, substr := range tt.wantContains {
				assert.Contains(t, path, substr)
			}

			assert.DirExists(t, filepath.Dir(path))
		})
	}
}

func TestCoveragePath_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "new", "nested", "coverage")
	directory := "internal/service"
	testName := "TestAuthorService_CreateAuthor"

	path := CoveragePath(baseDir, directory, testName)

	assert.DirExists(t, filepath.Dir(path))
	assert.Contains(t, path, testName+".cov")

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCoveragePath_DirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "coverage")
	directory := "internal/repository"
	testName := "TestBookRepository_FindByID"

	err := os.MkdirAll(filepath.Join(baseDir, "internal_repository"), 0755)
	require.NoError(t, err)

	path := CoveragePath(baseDir, directory, testName)

	assert.DirExists(t, filepath.Dir(path))
	assert.Contains(t, path, testName+".cov")
}
