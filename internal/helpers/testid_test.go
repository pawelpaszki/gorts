package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQualifyTestName(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		testName  string
		want      string
	}{
		{
			name:      "standard directory and test",
			directory: "test/integration",
			testName:  "TestAuthor_Create",
			want:      "test/integration/TestAuthor_Create",
		},
		{
			name:      "directory with leading ./",
			directory: "./test/e2e",
			testName:  "TestE2E_BookCRUD",
			want:      "test/e2e/TestE2E_BookCRUD",
		},
		{
			name:      "directory with trailing /",
			directory: "internal/handler/",
			testName:  "TestBookHandler_Get",
			want:      "internal/handler/TestBookHandler_Get",
		},
		{
			name:      "directory with both ./ and trailing /",
			directory: "./test/integration/",
			testName:  "TestReadingList_AddBook",
			want:      "test/integration/TestReadingList_AddBook",
		},
		{
			name:      "root directory",
			directory: ".",
			testName:  "TestMain",
			want:      "./TestMain",
		},
		{
			name:      "empty directory",
			directory: "",
			testName:  "TestValidate",
			want:      "/TestValidate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QualifyTestName(tt.directory, tt.testName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseQualifiedTest(t *testing.T) {
	tests := []struct {
		name     string
		test     string
		wantTest string
		wantDir  string
	}{
		{
			name:     "standard qualified name",
			test:     "test/integration/TestAuthor_Create",
			wantTest: "TestAuthor_Create",
			wantDir:  "test/integration",
		},
		{
			name:     "nested directory",
			test:     "internal/handler/book/TestHandler_Get",
			wantTest: "TestHandler_Get",
			wantDir:  "internal/handler/book",
		},
		{
			name:     "single directory",
			test:     "test/TestMain",
			wantTest: "TestMain",
			wantDir:  "test",
		},
		{
			name:     "no directory",
			test:     "TestStandalone",
			wantTest: "TestStandalone",
			wantDir:  "",
		},
		{
			name:     "root with dot",
			test:     "./TestRoot",
			wantTest: "TestRoot",
			wantDir:  ".",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotTest := ParseQualifiedTest(tt.test)
			assert.Equal(t, tt.wantTest, gotTest)
			assert.Equal(t, tt.wantDir, gotDir)
		})
	}
}

func TestDirectoryForGoTest(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		wantDir   string
	}{
		{
			name:      "standard directory",
			directory: "test/integration",
			wantDir:   "./test/integration/...",
		},
		{
			name:      "directory with ./ prefix",
			directory: "./internal/handler",
			wantDir:   "./internal/handler/...",
		},
		{
			name:      "root directory",
			directory: ".",
			wantDir:   "./...",
		},
		{
			name:      "empty string",
			directory: "",
			wantDir:   ".//...",
		},
		{
			name:      "nested directory",
			directory: "internal/service/book",
			wantDir:   "./internal/service/book/...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir := DirectoryForGoTest(tt.directory)
			assert.Equal(t, tt.wantDir, gotDir)
		})
	}
}

func TestMatchesRunAllPattern(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match go.mod",
			file:     "go.mod",
			patterns: []string{"go.mod"},
			want:     true,
		},
		{
			name:     "suffix match config.go",
			file:     "internal/config/config.go",
			patterns: []string{"config.go"},
			want:     true,
		},
		{
			name:     "wildcard *.go matches",
			file:     "pkg/stringutil/stringutil.go",
			patterns: []string{"*.go"},
			want:     true,
		},
		{
			name:     "wildcard go.* matches go.sum",
			file:     "go.sum",
			patterns: []string{"go.*"},
			want:     true,
		},
		{
			name:     "no match returns false",
			file:     "internal/handler/book_handler.go",
			patterns: []string{"go.mod", "go.sum"},
			want:     false,
		},
		{
			name:     "empty patterns returns false",
			file:     "any_file.go",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "multiple patterns first matches",
			file:     "Makefile",
			patterns: []string{"Makefile", "go.mod"},
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			file:     "go.sum",
			patterns: []string{"go.mod", "go.sum"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesRunAllPattern(tt.file, tt.patterns)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckRunAllTrigger(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles []string
		patterns     []string
		wantMatch    bool
		wantFile     string
	}{
		{
			name:         "trigger on go.mod",
			changedFiles: []string{"internal/model/book.go", "go.mod"},
			patterns:     []string{"go.mod"},
			wantMatch:    true,
			wantFile:     "go.mod",
		},
		{
			name:         "trigger on Makefile",
			changedFiles: []string{"Makefile", "README.md"},
			patterns:     []string{"Makefile", "go.mod"},
			wantMatch:    true,
			wantFile:     "Makefile",
		},
		{
			name:         "no trigger when no match",
			changedFiles: []string{"internal/handler/book_handler.go"},
			patterns:     []string{"go.mod", "go.sum"},
			wantMatch:    false,
			wantFile:     "",
		},
		{
			name:         "empty changed files returns no match",
			changedFiles: []string{},
			patterns:     []string{"go.mod"},
			wantMatch:    false,
			wantFile:     "",
		},
		{
			name:         "empty patterns returns no match",
			changedFiles: []string{"go.mod"},
			patterns:     []string{},
			wantMatch:    false,
			wantFile:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotFile := CheckRunAllTrigger(tt.changedFiles, tt.patterns)
			assert.Equal(t, tt.wantMatch, gotMatch)
			assert.Equal(t, tt.wantFile, gotFile)
		})
	}
}
