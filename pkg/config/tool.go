package config

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/viper"
)

func GetToolPath(application string, name string) (string, error) {
	toolPath := viper.GetString(fmt.Sprintf("%s.%s", application, name))
	if toolPath == "" {
		return "", fmt.Errorf("tool not found in the config file: %s.%s", application, name)
	}

	// The configured value may be an executable path or a bare command name
	// (e.g. "convert"): use it directly if it's an existing file, else fall back
	// to a PATH lookup so a bare name resolves.
	if _, err := os.Stat(toolPath); err == nil {
		return toolPath, nil
	}

	if resolved, err := exec.LookPath(toolPath); err == nil {
		return resolved, nil
	}

	return "", fmt.Errorf("tool %q (%s.%s) does not exist and was not found on PATH", toolPath, application, name)
}
