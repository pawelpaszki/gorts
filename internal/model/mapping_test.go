package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateCoverageMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping *CoverageMapping
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid mapping",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: map[string][]string{
					"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
				},
				TestToFiles: map[string][]string{
					"test/integration/TestBook_Create": {"internal/handler/book_handler.go"},
				},
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "valid mapping with function level data",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: map[string][]string{
					"internal/service/author_service.go": {"test/integration/TestAuthor_Create"},
				},
				TestToFiles: map[string][]string{
					"test/integration/TestAuthor_Create": {"internal/service/author_service.go"},
				},
				FunctionToTests: map[string][]string{
					"internal/service/author_service.go::CreateAuthor": {"test/integration/TestAuthor_Create"},
				},
				TestToFunctions: map[string][]string{
					"test/integration/TestAuthor_Create": {"internal/service/author_service.go::CreateAuthor"},
				},
				FunctionChecksums: map[string]string{
					"internal/service/author_service.go::CreateAuthor": "abc123",
				},
				Stats: MappingStats{
					TotalTests:     1,
					TotalFiles:     1,
					TotalFunctions: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "missing generated_at",
			mapping: &CoverageMapping{
				CommitSHA: "abc123def456",
				FileToTests: map[string][]string{
					"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
				},
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 1,
				},
			},
			wantErr: true,
			errMsg:  "missing generated_at",
		},
		{
			name: "missing commit_sha",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				FileToTests: map[string][]string{
					"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
				},
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 1,
				},
			},
			wantErr: true,
			errMsg:  "missing commit_sha",
		},
		{
			name: "empty file_to_tests mapping",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: map[string][]string{},
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 1,
				},
			},
			wantErr: true,
			errMsg:  "file_to_tests mapping is empty",
		},
		{
			name: "nil file_to_tests mapping",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: nil,
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 1,
				},
			},
			wantErr: true,
			errMsg:  "file_to_tests mapping is empty",
		},
		{
			name: "zero total_tests in stats",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: map[string][]string{
					"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
				},
				Stats: MappingStats{
					TotalTests: 0,
					TotalFiles: 1,
				},
			},
			wantErr: true,
			errMsg:  "invalid stats: total_tests is 0",
		},
		{
			name: "zero total_files in stats",
			mapping: &CoverageMapping{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				FileToTests: map[string][]string{
					"internal/handler/book_handler.go": {"test/integration/TestBook_Create"},
				},
				Stats: MappingStats{
					TotalTests: 1,
					TotalFiles: 0,
				},
			},
			wantErr: true,
			errMsg:  "invalid stats: total_files is 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoverageMapping(tt.mapping)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
