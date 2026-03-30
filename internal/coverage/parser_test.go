package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFilePath(t *testing.T) {
	tests := []struct {
		name       string
		fullPath   string
		modulePath string
		want       string
	}{
		{
			name:       "strip standard module path",
			fullPath:   "github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/handler/book_handler.go",
		},
		{
			name:       "strip GOPATH prefix",
			fullPath:   "/go/src/github.com/pawelpaszki/gorts-demo/internal/service/author_service.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/service/author_service.go",
		},
		{
			name:       "strip container mount prefix",
			fullPath:   "/app/github.com/pawelpaszki/gorts-demo/internal/repository/book_repo.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/repository/book_repo.go",
		},
		{
			name:       "strip local GOPATH prefix",
			fullPath:   "/home/user/go/src/github.com/pawelpaszki/gorts-demo/pkg/validator/validator.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "pkg/validator/validator.go",
		},
		{
			name:       "handle nested directories",
			fullPath:   "github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/middleware/auth.go",
		},
		{
			name:       "return unchanged if module not found",
			fullPath:   "/other/path/config.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "/other/path/config.go",
		},
		{
			name:       "handle empty module path",
			fullPath:   "github.com/pawelpaszki/gorts-demo/internal/model/book.go",
			modulePath: "",
			want:       "github.com/pawelpaszki/gorts-demo/internal/model/book.go",
		},
		{
			name:       "handle file at module root",
			fullPath:   "github.com/pawelpaszki/gorts-demo/main.go",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeFilePath(tt.fullPath, tt.modulePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTextCoverage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "parse single covered file",
			content: "mode: set\ngithub.com/pawelpaszki/gorts-demo/internal/model/book.go:22.1,42.2 1 1",
			want:    []string{"github.com/pawelpaszki/gorts-demo/internal/model/book.go"},
			wantErr: false,
		},
		{
			name: "parse multiple covered files",
			content: `mode: set
github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go:10.1,15.2 3 1
github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go:20.1,25.2 2 1
github.com/pawelpaszki/gorts-demo/internal/service/book_service.go:5.1,10.2 1 1
github.com/pawelpaszki/gorts-demo/internal/model/author.go:1.1,5.2 1 1`,
			want: []string{
				"github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go",
				"github.com/pawelpaszki/gorts-demo/internal/service/book_service.go",
				"github.com/pawelpaszki/gorts-demo/internal/model/author.go",
			},
			wantErr: false,
		},
		{
			name:    "skip uncovered lines (count=0)",
			content: "github.com/pawelpaszki/gorts-demo/internal/config/config.go:1.1,2.2 1 0",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "skip mode line",
			content: "mode: set\n...",
			want:    []string{},
			wantErr: false,
		},
		{
			name: "skip empty lines",
			content: `mode: set
github.com/pawelpaszki/gorts-demo/internal/handler/author_handler.go:10.1,15.2 3 1
github.com/pawelpaszki/gorts-demo/internal/handler/author_handler.go:20.1,25.2 2 1
github.com/pawelpaszki/gorts-demo/internal/service/author_service.go:5.1,10.2 1 1

github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go:1.1,5.2 1 1
`,
			want: []string{
				"github.com/pawelpaszki/gorts-demo/internal/handler/author_handler.go",
				"github.com/pawelpaszki/gorts-demo/internal/service/author_service.go",
				"github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go",
			},
			wantErr: false,
		},
		{
			name: "deduplicate repeated files",
			content: `mode: set
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:1.1,5.2 1 1
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:10.1,15.2 1 1
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:20.1,25.2 1 1
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:30.1,35.2 1 1`,
			want:    []string{"github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go"},
			wantErr: false,
		},
		{
			name:    "skip malformed line with less than 3 fields",
			content: "mode: set\ngithub.com/pawelpaszki/gorts-demo/internal/handler/health_handler.go:1.1,2.2 1",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "skip line without colon",
			content: "mode: set\ninvalidline 1 1",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "return empty for empty file",
			content: "",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "return empty for mode-only file",
			content: "mode: set",
			want:    []string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "coverage.out")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)
			got, err := parseTextCoverage(tmpFile)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestParseTextCoverage_NonExistentFile(t *testing.T) {
	_, err := parseTextCoverage("/nonexistent/path/coverage.out")
	assert.Error(t, err)
}

func TestFindCoverageFiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) []string
		wantEmpty bool
		wantErr   bool
	}{
		{
			name: "detect binary format with covmeta files",
			setup: func(t *testing.T, dir string) []string {
				err := os.WriteFile(filepath.Join(dir, "covmeta.abc123def"), []byte{}, 0644)
				require.NoError(t, err)
				return []string{dir}
			},
		},
		{
			name: "find .out files recursively",
			setup: func(t *testing.T, dir string) []string {
				subdir := filepath.Join(dir, "test_e2e")
				err := os.MkdirAll(subdir, 0755)
				require.NoError(t, err)

				file1 := filepath.Join(dir, "TestE2E_CreateBook.out")
				file2 := filepath.Join(subdir, "TestE2E_GetAuthor.out")
				err = os.WriteFile(file1, []byte("mode: set"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(file2, []byte("mode: set"), 0644)
				require.NoError(t, err)

				return []string{file1, file2}
			},
		},
		{
			name: "return empty for empty directory",
			setup: func(t *testing.T, dir string) []string {
				return nil
			},
			wantEmpty: true,
		},
		{
			name: "ignore non-.out files",
			setup: func(t *testing.T, dir string) []string {
				err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644)
				require.NoError(t, err)
				return nil
			},
			wantEmpty: true,
		},
		{
			name: "prefer binary format over .out files",
			setup: func(t *testing.T, dir string) []string {
				err := os.WriteFile(filepath.Join(dir, "covmeta.abc123"), []byte{}, 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(dir, "TestE2E_BookCRUD.out"), []byte("mode: set"), 0644)
				require.NoError(t, err)
				return []string{dir}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			want := tt.setup(t, dir)

			got, err := FindCoverageFiles(dir)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.ElementsMatch(t, want, got)
			}
		})
	}
}

func TestFindCoverageFiles_NonExistentDir(t *testing.T) {
	_, err := FindCoverageFiles("/nonexistent/path")
	assert.Error(t, err)
}

func TestParseCoverageFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string
		want      []string
		wantEmpty bool
		wantErr   bool
	}{
		{
			name: "delegate to parseTextCoverage for files",
			setup: func(t *testing.T, dir string) string {
				tmpFile := filepath.Join(dir, "coverage.out")
				content := "mode: set\ngithub.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go:1.1,2.2 1 1"
				err := os.WriteFile(tmpFile, []byte(content), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			want: []string{"github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go"},
		},
		{
			name: "delegate to parseBinaryCoverage for directories",
			setup: func(t *testing.T, dir string) string {
				return dir
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)

			got, err := ParseCoverageFile(path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.ElementsMatch(t, tt.want, got)
			}
		})
	}
}

func TestParseCoverageFile_NonExistentPath(t *testing.T) {
	_, err := ParseCoverageFile("/nonexistent/path")
	assert.Error(t, err)
}
