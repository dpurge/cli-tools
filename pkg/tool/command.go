package tool

import (
	"log"
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

// RunCmdLogged runs a configured tool like RunCmd and additionally logs the
// command's combined output when it is non-empty (e.g. tool warnings). It
// folds the "if len(output) > 0 { log.Println(output) }" idiom repeated by
// callers that run a tool purely for its side effects. Use RunCmd directly
// when the output is consumed as data, so it is not echoed to the log.
func RunCmdLogged(group string, tool string, args ...string) (string, error) {
	output, err := RunCmd(group, tool, args...)
	if err != nil {
		return "", err
	}
	if len(output) > 0 {
		log.Println(output)
	}
	return output, nil
}
