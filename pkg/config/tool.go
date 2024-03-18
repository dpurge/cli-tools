package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func GetToolPath(application string, name string) (string, error) {
	toolPath := viper.GetString(fmt.Sprintf("%s.%s", application, name))
	if toolPath == "" {
		return "", fmt.Errorf("tool not found in the config file: %s.%s", application, name)
	}

	if _, err := os.Stat(toolPath); err != nil && os.IsNotExist(err) {
		return "", fmt.Errorf("tool does not exist: %s", toolPath)
	}

	return toolPath, nil
}
