package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSelectedTest_Qualified(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		testName  string
		want      string
	}{
		{
			name:      "standard integration test",
			directory: "test/integration",
			testName:  "TestAuthor_Create",
			want:      "test/integration/TestAuthor_Create",
		},
		{
			name:      "e2e test",
			directory: "test/e2e",
			testName:  "TestE2E_BookCRUD",
			want:      "test/e2e/TestE2E_BookCRUD",
		},
		{
			name:      "nested directory",
			directory: "internal/service/book",
			testName:  "TestBookService_Create",
			want:      "internal/service/book/TestBookService_Create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SelectedTest{
				Directory: tt.directory,
				TestName:  tt.testName,
			}
			got := s.Qualified()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSelectedTest_ForGoTestRun(t *testing.T) {
	tests := []struct {
		name     string
		testName string
		want     string
	}{
		{
			name:     "simple test name",
			testName: "TestAuthor_Create",
			want:     "^TestAuthor_Create$",
		},
		{
			name:     "test with underscores",
			testName: "TestBook_Validate_ISBN",
			want:     "^TestBook_Validate_ISBN$",
		},
		{
			name:     "e2e test name",
			testName: "TestE2E_CreateAndGetBook",
			want:     "^TestE2E_CreateAndGetBook$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SelectedTest{
				Directory: "test/integration",
				TestName:  tt.testName,
			}
			got := s.ForGoTestRun()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateSelection(t *testing.T) {
	tests := []struct {
		name      string
		selection *Selection
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid selection",
			selection: &Selection{
				GeneratedAt: time.Now(),
				FromCommit:  "abc123",
				ToCommit:    "def456",
			},
			wantErr: false,
		},
		{
			name: "missing generated_at",
			selection: &Selection{
				FromCommit: "abc123",
				ToCommit:   "def456",
			},
			wantErr: true,
			errMsg:  "missing generated_at",
		},
		{
			name: "missing from_commit",
			selection: &Selection{
				GeneratedAt: time.Now(),
				ToCommit:    "def456",
			},
			wantErr: true,
			errMsg:  "missing from_commit",
		},
		{
			name: "missing to_commit",
			selection: &Selection{
				GeneratedAt: time.Now(),
				FromCommit:  "abc123",
			},
			wantErr: true,
			errMsg:  "missing to_commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSelection(tt.selection)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
