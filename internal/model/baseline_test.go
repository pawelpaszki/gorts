package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateBaselineManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *BaselineManifest
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid baseline manifest",
			manifest: &BaselineManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuiteResults: []TestSuiteResult{
					{
						Directory: "test/integration",
						TestResults: []TestResult{
							{
								Directory:  "test/integration",
								TestName:   "TestAuthor_Create",
								Status:     "pass",
								DurationMs: 100,
							},
						},
						Summary: Summary{
							Total:  1,
							Passed: 1,
						},
					},
				},
				Summary: Summary{
					Total:      1,
					Passed:     1,
					DurationMs: 100,
				},
			},
			wantErr: false,
		},
		{
			name: "valid baseline with multiple suites",
			manifest: &BaselineManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuiteResults: []TestSuiteResult{
					{
						Directory: "test/integration",
						TestResults: []TestResult{
							{TestName: "TestAuthor_Create", Status: "pass"},
							{TestName: "TestBook_Create", Status: "pass"},
						},
						Summary: Summary{Total: 2, Passed: 2},
					},
					{
						Directory: "test/e2e",
						TestResults: []TestResult{
							{TestName: "TestE2E_FullFlow", Status: "pass"},
						},
						Summary: Summary{Total: 1, Passed: 1},
					},
				},
				Summary: Summary{Total: 3, Passed: 3},
			},
			wantErr: false,
		},
		{
			name: "valid baseline with flaky test",
			manifest: &BaselineManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuiteResults: []TestSuiteResult{
					{
						Directory: "test/integration",
						TestResults: []TestResult{
							{
								TestName: "TestReadingList_Add",
								Status:   "pass",
								Retries:  2,
								Flaky:    true,
							},
						},
						Summary: Summary{Total: 1, Passed: 1, Flaky: 1},
					},
				},
				Summary: Summary{Total: 1, Passed: 1, Flaky: 1},
			},
			wantErr: false,
		},
		{
			name: "missing generated_at",
			manifest: &BaselineManifest{
				CommitSHA: "abc123def456",
				TestSuiteResults: []TestSuiteResult{
					{Directory: "test/integration"},
				},
				Summary: Summary{Total: 1},
			},
			wantErr: true,
			errMsg:  "missing generated_at",
		},
		{
			name: "missing commit_sha",
			manifest: &BaselineManifest{
				GeneratedAt: time.Now(),
				TestSuiteResults: []TestSuiteResult{
					{Directory: "test/integration"},
				},
				Summary: Summary{Total: 1},
			},
			wantErr: true,
			errMsg:  "missing commit_sha",
		},
		{
			name: "empty test_suite_results",
			manifest: &BaselineManifest{
				GeneratedAt:      time.Now(),
				CommitSHA:        "abc123def456",
				TestSuiteResults: []TestSuiteResult{},
				Summary:          Summary{Total: 1},
			},
			wantErr: true,
			errMsg:  "missing test_suite_results",
		},
		{
			name: "nil test_suite_results",
			manifest: &BaselineManifest{
				GeneratedAt:      time.Now(),
				CommitSHA:        "abc123def456",
				TestSuiteResults: nil,
				Summary:          Summary{Total: 1},
			},
			wantErr: true,
			errMsg:  "missing test_suite_results",
		},
		{
			name: "zero total in summary",
			manifest: &BaselineManifest{
				GeneratedAt: time.Now(),
				CommitSHA:   "abc123def456",
				TestSuiteResults: []TestSuiteResult{
					{Directory: "test/integration"},
				},
				Summary: Summary{Total: 0},
			},
			wantErr: true,
			errMsg:  "summary is empty: no tests were run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBaselineManifest(tt.manifest)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
