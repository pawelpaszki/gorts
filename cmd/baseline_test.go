package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandHookCommand(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		dir          string
		testName     string
		coveragePath string
		want         string
	}{
		{
			name:         "expand all placeholders",
			cmd:          "echo {{DIR}} {{TEST}} {{COVERAGE_PATH}}",
			dir:          "test/integration",
			testName:     "TestAuthor_Create",
			coveragePath: "/tmp/coverage",
			want:         "echo test/integration TestAuthor_Create /tmp/coverage",
		},
		{
			name:         "no placeholders",
			cmd:          "echo hello",
			dir:          "test",
			testName:     "Test",
			coveragePath: "/cov",
			want:         "echo hello",
		},
		{
			name:         "multiple same placeholders",
			cmd:          "{{TEST}} - {{TEST}}",
			dir:          "dir",
			testName:     "TestFoo",
			coveragePath: "",
			want:         "TestFoo - TestFoo",
		},
		{
			name:         "only dir placeholder",
			cmd:          "cd {{DIR}} && make test",
			dir:          "test/e2e",
			testName:     "Test",
			coveragePath: "",
			want:         "cd test/e2e && make test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHookCommand(tt.cmd, tt.dir, tt.testName, tt.coveragePath)
			assert.Equal(t, tt.want, got)
		})
	}
}
