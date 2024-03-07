package cfg

import (
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

	viper.AddConfigPath(filepath.Join(homedir, ".config", "cli-tools"))
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
}
