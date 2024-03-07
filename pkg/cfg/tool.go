package cfg

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func GetToolPath(application string, name string) (string, error) {
	toolPath := viper.GetString(fmt.Sprintf("%s.%s", application, name))
	if toolPath == "" {
		log.Fatalf("Tool not found in the config file: %s.%s", application, name)
	}
	return toolPath, nil
}
