package coverage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFuncOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []FunctionCoverage
		wantErr bool
	}{
		{
			name: "parse single function coverage",
			output: `github.com/pawelpaszki/gorts-demo/internal/model/book.go:22:	Validate	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/model/book.go", LineNumber: 22, FunctionName: "Validate", Coverage: 100.0},
			},
		},
		{
			name: "parse multiple functions in single file",
			output: `github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go:21:	Validate	100.0%
github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go:35:	AddBook	100.0%
github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go:46:	RemoveBook	100.0%
github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go:57:	ContainsBook	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go", LineNumber: 21, FunctionName: "Validate", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go", LineNumber: 35, FunctionName: "AddBook", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go", LineNumber: 46, FunctionName: "RemoveBook", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go", LineNumber: 57, FunctionName: "ContainsBook", Coverage: 100.0},
			},
		},
		{
			name: "parse functions from multiple files",
			output: `github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go:10:	HandleGetBook	100.0%
github.com/pawelpaszki/gorts-demo/internal/handler/author_handler.go:15:	HandleGetAuthor	100.0%
github.com/pawelpaszki/gorts-demo/internal/service/book_service.go:20:	CreateBook	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go", LineNumber: 10, FunctionName: "HandleGetBook", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/handler/author_handler.go", LineNumber: 15, FunctionName: "HandleGetAuthor", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/service/book_service.go", LineNumber: 20, FunctionName: "CreateBook", Coverage: 100.0},
			},
		},
		{
			name: "skip total line",
			output: `github.com/pawelpaszki/gorts-demo/pkg/stringutil/stringutil.go:10:	Truncate	100.0%
github.com/pawelpaszki/gorts-demo/pkg/stringutil/stringutil.go:23:	Slugify	100.0%
total:								(statements)	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/pkg/stringutil/stringutil.go", LineNumber: 10, FunctionName: "Truncate", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/pkg/stringutil/stringutil.go", LineNumber: 23, FunctionName: "Slugify", Coverage: 100.0},
			},
		},
		{
			name:   "return empty slice for empty output",
			output: "",
			want:   nil,
		},
		{
			name:   "return empty slice for only total line",
			output: "total:\t\t\t\t\t\t\t(statements)\t85.0%\n",
			want:   nil,
		},
		{
			name: "skip lines without proper colon format",
			output: `malformed line without colons
github.com/pawelpaszki/gorts-demo/internal/config/config.go:10:	LoadConfig	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/config/config.go", LineNumber: 10, FunctionName: "LoadConfig", Coverage: 100.0},
			},
		},
		{
			name: "handle coverage with partial percentage",
			output: `github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:43:	NewInMemoryUserStore	100.0%
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:51:	AddUser	66.7%
github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go:57:	Authenticate	50.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go", LineNumber: 43, FunctionName: "NewInMemoryUserStore", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go", LineNumber: 51, FunctionName: "AddUser", Coverage: 66.7},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/middleware/auth.go", LineNumber: 57, FunctionName: "Authenticate", Coverage: 50.0},
			},
		},
		{
			name: "skip zero coverage functions",
			output: `github.com/pawelpaszki/gorts-demo/internal/repository/book_repo.go:15:	FindByID	100.0%
github.com/pawelpaszki/gorts-demo/internal/repository/book_repo.go:25:	Delete	0.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/repository/book_repo.go", LineNumber: 15, FunctionName: "FindByID", Coverage: 100.0},
			},
		},
		{
			name: "handle plain functions",
			output: `github.com/pawelpaszki/gorts-demo/cmd/server/main.go:10:	main	100.0%
github.com/pawelpaszki/gorts-demo/pkg/validator/validator.go:15:	ValidateEmail	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/cmd/server/main.go", LineNumber: 10, FunctionName: "main", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/pkg/validator/validator.go", LineNumber: 15, FunctionName: "ValidateEmail", Coverage: 100.0},
			},
		},
		{
			name: "skip lines with insufficient fields after colon",
			output: `github.com/pawelpaszki/gorts-demo/internal/service/book_service.go:10:	GetBook
github.com/pawelpaszki/gorts-demo/internal/service/book_service.go:20:	CreateBook	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/service/book_service.go", LineNumber: 20, FunctionName: "CreateBook", Coverage: 100.0},
			},
		},
		{
			name: "handle whitespace and empty lines",
			output: `github.com/pawelpaszki/gorts-demo/internal/handler/health_handler.go:10:	HandleHealth	100.0%

github.com/pawelpaszki/gorts-demo/internal/handler/health_handler.go:20:	HandleReady	100.0%
`,
			want: []FunctionCoverage{
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/handler/health_handler.go", LineNumber: 10, FunctionName: "HandleHealth", Coverage: 100.0},
				{FilePath: "github.com/pawelpaszki/gorts-demo/internal/handler/health_handler.go", LineNumber: 20, FunctionName: "HandleReady", Coverage: 100.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFuncOutput(tt.output)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.want == nil {
				assert.Empty(t, got)
				return
			}

			assert.Len(t, got, len(tt.want))
			for i, wantFunc := range tt.want {
				assert.Equal(t, wantFunc.FilePath, got[i].FilePath, "FilePath mismatch at index %d", i)
				assert.Equal(t, wantFunc.LineNumber, got[i].LineNumber, "LineNumber mismatch at index %d", i)
				assert.Equal(t, wantFunc.FunctionName, got[i].FunctionName, "FunctionName mismatch at index %d", i)
				assert.InDelta(t, wantFunc.Coverage, got[i].Coverage, 0.1, "Coverage mismatch at index %d", i)
			}
		})
	}
}

func TestQualifyFunction(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		funcName   string
		modulePath string
		want       string
	}{
		{
			name:       "qualify plain function",
			filePath:   "github.com/pawelpaszki/gorts-demo/internal/handler/book_handler.go",
			funcName:   "HandleGetBook",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/handler/book_handler.go::HandleGetBook",
		},
		{
			name:       "qualify method with pointer receiver",
			filePath:   "github.com/pawelpaszki/gorts-demo/internal/model/book.go",
			funcName:   "(*Book).Validate",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/model/book.go::(*Book).Validate",
		},
		{
			name:       "qualify method with value receiver",
			filePath:   "github.com/pawelpaszki/gorts-demo/internal/model/reading_list.go",
			funcName:   "(ReadingList).Slug",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/model/reading_list.go::(ReadingList).Slug",
		},
		{
			name:       "qualify nested path function",
			filePath:   "github.com/pawelpaszki/gorts-demo/internal/repository/author_repo.go",
			funcName:   "FindAll",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/repository/author_repo.go::FindAll",
		},
		{
			name:       "qualify root level function",
			filePath:   "github.com/pawelpaszki/gorts-demo/main.go",
			funcName:   "main",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "main.go::main",
		},
		{
			name:       "qualify init function",
			filePath:   "github.com/pawelpaszki/gorts-demo/internal/config/config.go",
			funcName:   "init",
			modulePath: "github.com/pawelpaszki/gorts-demo",
			want:       "internal/config/config.go::init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QualifyFunction(tt.filePath, tt.funcName, tt.modulePath)
			assert.Equal(t, tt.want, got)
		})
	}
}
