package exec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("successful command returns stdout", func(t *testing.T) {
		stdout, stderr, err := Run("", "echo", "hello world")

		require.NoError(t, err)
		assert.Equal(t, "hello world\n", stdout)
		assert.Empty(t, stderr)
	})

	t.Run("command respects working directory", func(t *testing.T) {
		stdout, _, err := Run("/tmp", "pwd")

		require.NoError(t, err)
		assert.Contains(t, stdout, "/tmp")
	})

	t.Run("failed command returns error", func(t *testing.T) {
		_, _, err := Run("", "ls", "/nonexistent/path/that/does/not/exist")

		assert.Error(t, err)
	})

	t.Run("command with stderr output", func(t *testing.T) {
		// Using sh -c to redirect echo to stderr
		_, stderr, err := Run("", "sh", "-c", "echo error >&2")

		require.NoError(t, err)
		assert.Equal(t, "error\n", stderr)
	})

	t.Run("command not found returns error", func(t *testing.T) {
		_, _, err := Run("", "nonexistent_command_12345")

		assert.Error(t, err)
	})

	t.Run("command with multiple arguments", func(t *testing.T) {
		stdout, _, err := Run("", "echo", "-n", "no newline")

		require.NoError(t, err)
		assert.Equal(t, "no newline", stdout)
		assert.False(t, strings.HasSuffix(stdout, "\n"))
	})
}
