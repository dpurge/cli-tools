package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func ReadConfig() {

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configDir := filepath.Join(homedir, ".config", "cli-tools")
	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	err = viper.ReadInConfig()
	if err != nil {
		// A missing config file is not fatal: commands that need no
		// configured values (e.g. --version, --help) still run, and
		// commands that do (via config.GetToolPath) surface a clear error
		// at their point of use instead of blocking every invocation.
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			log.Printf("warning: no config file found in %s; continuing without it", configDir)
			return
		}
		// A config file that exists but is unreadable/malformed is a real
		// problem the user must fix.
		log.Fatal(err)
	}
}
