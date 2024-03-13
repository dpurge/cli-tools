package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func GetToolPath(application string, name string) (string, error) {
	var err error

	toolPath := viper.GetString(fmt.Sprintf("%s.%s", application, name))
	if toolPath == "" {
		err = fmt.Errorf("tool not found in the config file: %s.%s", application, name)
	} else {
		_, err = os.Stat(toolPath)
	}

	return toolPath, err
}
