package cmd

import (
	"testing"

	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestFilterGoFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name:  "only go files",
			files: []string{"main.go", "handler.go", "model.go"},
			want:  []string{"main.go", "handler.go", "model.go"},
		},
		{
			name:  "mixed files",
			files: []string{"main.go", "README.md", "handler.go", "Makefile"},
			want:  []string{"main.go", "handler.go"},
		},
		{
			name:  "no go files",
			files: []string{"README.md", "Makefile", "config.yaml"},
			want:  nil,
		},
		{
			name:  "empty input",
			files: []string{},
			want:  nil,
		},
		{
			name:  "test files included",
			files: []string{"handler.go", "handler_test.go"},
			want:  []string{"handler.go", "handler_test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterGoFiles(tt.files)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildBaselineDirs(t *testing.T) {
	tests := []struct {
		name     string
		baseline *model.BaselineManifest
		want     map[string]bool
	}{
		{
			name: "single suite",
			baseline: &model.BaselineManifest{
				TestSuiteResults: []model.TestSuiteResult{
					{Directory: "test/integration"},
				},
			},
			want: map[string]bool{"test/integration": true},
		},
		{
			name: "multiple suites",
			baseline: &model.BaselineManifest{
				TestSuiteResults: []model.TestSuiteResult{
					{Directory: "test/integration"},
					{Directory: "test/e2e"},
					{Directory: "internal/model"},
				},
			},
			want: map[string]bool{
				"test/integration": true,
				"test/e2e":         true,
				"internal/model":   true,
			},
		},
		{
			name: "empty baseline",
			baseline: &model.BaselineManifest{
				TestSuiteResults: []model.TestSuiteResult{},
			},
			want: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBaselineDirs(tt.baseline)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCategorizeChangedFiles(t *testing.T) {
	baselineDirs := map[string]bool{
		"test/integration": true,
		"test/e2e":         true,
	}

	tests := []struct {
		name                    string
		changedFiles            []string
		wantSourceFiles         []string
		wantInScopeTestFiles    []string
		wantOutOfScopeTestFiles []string
	}{
		{
			name:                    "only source files",
			changedFiles:            []string{"internal/handler/book.go", "internal/model/author.go"},
			wantSourceFiles:         []string{"internal/handler/book.go", "internal/model/author.go"},
			wantInScopeTestFiles:    nil,
			wantOutOfScopeTestFiles: nil,
		},
		{
			name:                    "in-scope test files",
			changedFiles:            []string{"test/integration/author_test.go", "test/e2e/book_test.go"},
			wantSourceFiles:         nil,
			wantInScopeTestFiles:    []string{"test/integration/author_test.go", "test/e2e/book_test.go"},
			wantOutOfScopeTestFiles: nil,
		},
		{
			name:                    "out-of-scope test files",
			changedFiles:            []string{"internal/model/book_test.go", "pkg/util/util_test.go"},
			wantSourceFiles:         nil,
			wantInScopeTestFiles:    nil,
			wantOutOfScopeTestFiles: []string{"internal/model/book_test.go", "pkg/util/util_test.go"},
		},
		{
			name:                    "mixed files",
			changedFiles:            []string{"internal/handler/book.go", "test/integration/author_test.go", "internal/model/book_test.go"},
			wantSourceFiles:         []string{"internal/handler/book.go"},
			wantInScopeTestFiles:    []string{"test/integration/author_test.go"},
			wantOutOfScopeTestFiles: []string{"internal/model/book_test.go"},
		},
		{
			name:                    "empty input",
			changedFiles:            []string{},
			wantSourceFiles:         nil,
			wantInScopeTestFiles:    nil,
			wantOutOfScopeTestFiles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, inScope, outOfScope := categorizeChangedFiles(tt.changedFiles, baselineDirs)
			assert.Equal(t, tt.wantSourceFiles, source)
			assert.Equal(t, tt.wantInScopeTestFiles, inScope)
			assert.Equal(t, tt.wantOutOfScopeTestFiles, outOfScope)
		})
	}
}

func TestCalculateReductionPercent(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		selected int
		want     float64
	}{
		{
			name:     "50% reduction",
			total:    100,
			selected: 50,
			want:     50.0,
		},
		{
			name:     "no reduction",
			total:    100,
			selected: 100,
			want:     0.0,
		},
		{
			name:     "full reduction",
			total:    100,
			selected: 0,
			want:     100.0,
		},
		{
			name:     "zero total",
			total:    0,
			selected: 0,
			want:     0.0,
		},
		{
			name:     "negative total",
			total:    -10,
			selected: 5,
			want:     0.0,
		},
		{
			name:     "75% reduction",
			total:    40,
			selected: 10,
			want:     75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateReductionPercent(tt.total, tt.selected)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGoTestList(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		directory string
		want      []string
	}{
		{
			name: "standard output",
			output: `TestAuthor_Create
TestAuthor_Update
TestBook_Create
ok  	github.com/example/repo/test/integration	0.005s`,
			directory: "test/integration",
			want: []string{
				"test/integration/TestAuthor_Create",
				"test/integration/TestAuthor_Update",
				"test/integration/TestBook_Create",
			},
		},
		{
			name: "with benchmarks and examples",
			output: `TestMain
BenchmarkProcess
ExampleHandler
FuzzInput
ok  	github.com/example/repo	0.001s`,
			directory: ".",
			want: []string{
				"./TestMain",
				"./BenchmarkProcess",
				"./ExampleHandler",
				"./FuzzInput",
			},
		},
		{
			name:      "no test files",
			output:    "?   	github.com/example/repo	[no test files]",
			directory: "pkg/util",
			want:      nil,
		},
		{
			name:      "empty output",
			output:    "",
			directory: "test",
			want:      nil,
		},
		{
			name: "filters non-test lines",
			output: `TestFoo
some random output
TestBar
ok  	done`,
			directory: "test",
			want: []string{
				"test/TestFoo",
				"test/TestBar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGoTestList(tt.output, tt.directory)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildSelectedTestsSlice(t *testing.T) {
	tests := []struct {
		name             string
		selectedTestsMap map[string]bool
		wantLen          int
	}{
		{
			name: "multiple tests",
			selectedTestsMap: map[string]bool{
				"test/integration/TestAuthor_Create": true,
				"test/integration/TestBook_Create":   true,
				"test/e2e/TestE2E_Flow":              true,
			},
			wantLen: 3,
		},
		{
			name:             "empty map",
			selectedTestsMap: map[string]bool{},
			wantLen:          0,
		},
		{
			name: "single test",
			selectedTestsMap: map[string]bool{
				"test/integration/TestAuthor_Create": true,
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSelectedTestsSlice(tt.selectedTestsMap)
			assert.Len(t, got, tt.wantLen)
		})
	}
}
