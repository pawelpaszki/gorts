package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitCsv(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single directory",
			input: "test/integration",
			want:  []string{"test/integration"},
		},
		{
			name:  "multiple directories",
			input: "test/integration,test/e2e,test/unit",
			want:  []string{"test/integration", "test/e2e", "test/unit"},
		},
		{
			name:  "directories with spaces",
			input: "test/integration , test/e2e , test/unit",
			want:  []string{"test/integration", "test/e2e", "test/unit"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "empty values filtered",
			input: "test/integration,,test/e2e,",
			want:  []string{"test/integration", "test/e2e"},
		},
		{
			name:  "only commas",
			input: ",,,",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCsv(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
