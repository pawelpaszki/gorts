package cmd

import (
	"testing"

	"github.com/pawelpaszki/gorts/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{
			name:  "no duplicates",
			items: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "with duplicates",
			items: []string{"a", "b", "a", "c", "b", "a"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "all duplicates",
			items: []string{"a", "a", "a"},
			want:  []string{"a"},
		},
		{
			name:  "empty input",
			items: []string{},
			want:  []string{},
		},
		{
			name:  "single item",
			items: []string{"only"},
			want:  []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicate(tt.items)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  []string
	}{
		{
			name:  "append new item",
			slice: []string{"a", "b"},
			item:  "c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "item already exists",
			slice: []string{"a", "b", "c"},
			item:  "b",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "append to empty slice",
			slice: []string{},
			item:  "a",
			want:  []string{"a"},
		},
		{
			name:  "append to nil slice",
			slice: nil,
			item:  "a",
			want:  []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUnique(tt.slice, tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name        string
		fileToTests map[string][]string
		testToFiles map[string][]string
		want        model.MappingStats
	}{
		{
			name: "standard mapping",
			fileToTests: map[string][]string{
				"file1.go": {"test1", "test2"},
				"file2.go": {"test1"},
			},
			testToFiles: map[string][]string{
				"test1": {"file1.go", "file2.go"},
				"test2": {"file1.go"},
			},
			want: model.MappingStats{
				TotalTests:      2,
				TotalFiles:      2,
				AvgFilesPerTest: 1.5,
				AvgTestsPerFile: 1.5,
			},
		},
		{
			name:        "empty mapping",
			fileToTests: map[string][]string{},
			testToFiles: map[string][]string{},
			want: model.MappingStats{
				TotalTests:      0,
				TotalFiles:      0,
				AvgFilesPerTest: 0,
				AvgTestsPerFile: 0,
			},
		},
		{
			name: "single test single file",
			fileToTests: map[string][]string{
				"handler.go": {"TestHandler"},
			},
			testToFiles: map[string][]string{
				"TestHandler": {"handler.go"},
			},
			want: model.MappingStats{
				TotalTests:      1,
				TotalFiles:      1,
				AvgFilesPerTest: 1.0,
				AvgTestsPerFile: 1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateStats(tt.fileToTests, tt.testToFiles)
			assert.Equal(t, tt.want.TotalTests, got.TotalTests)
			assert.Equal(t, tt.want.TotalFiles, got.TotalFiles)
			assert.Equal(t, tt.want.AvgFilesPerTest, got.AvgFilesPerTest)
			assert.Equal(t, tt.want.AvgTestsPerFile, got.AvgTestsPerFile)
		})
	}
}
