package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateTestManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *TestManifest
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid test manifest",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuites: []TestSuite{
					{
						Directory: "test/integration",
						Tests:     []string{"TestAuthor_Create", "TestBook_Create"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid manifest with multiple suites",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuites: []TestSuite{
					{
						Directory: "test/integration",
						Tests:     []string{"TestAuthor_Create", "TestBook_Create"},
					},
					{
						Directory: "test/e2e",
						Tests:     []string{"TestE2E_FullFlow", "TestE2E_BookCRUD"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid manifest with empty tests in suite",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuites: []TestSuite{
					{
						Directory: "test/integration",
						Tests:     []string{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing generated_at",
			manifest: &TestManifest{
				CommitSHA: "abc123def456",
				TestSuites: []TestSuite{
					{Directory: "test/integration"},
				},
			},
			wantErr: true,
			errMsg:  "missing generated_at",
		},
		{
			name: "missing commit_sha",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				TestSuites: []TestSuite{
					{Directory: "test/integration"},
				},
			},
			wantErr: true,
			errMsg:  "missing commit_sha",
		},
		{
			name: "empty test_suites",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuites:  []TestSuite{},
			},
			wantErr: true,
			errMsg:  "missing test_suites",
		},
		{
			name: "nil test_suites",
			manifest: &TestManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuites:  nil,
			},
			wantErr: true,
			errMsg:  "missing test_suites",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTestManifest(tt.manifest)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
