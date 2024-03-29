package tool

import (
	"os/exec"

	"github.com/dpurge/cli-tools/pkg/config"
)

func RunCmd(group string, tool string, args ...string) (string, error) {
	toolPath, err := config.GetToolPath(group, tool)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(toolPath, args...)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(buf[:]), nil
}
