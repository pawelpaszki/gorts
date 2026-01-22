package exec

import (
	"bytes"
	"os/exec"
)

// cmd.Dir is important to get list of test names
// this command needs to be executed from the same dir as the target module
func Run(dir, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
