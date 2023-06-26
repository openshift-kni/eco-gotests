package shell

import (
	"os/exec"
)

// ExecuteShellCmd function executes a shell command.
func ExecuteShellCmd(command string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	return output, nil
}
