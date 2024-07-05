package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// ConfigRepository is a struct that holds the configuration for the repository (database)
type Config struct {
	DbDSN string `yaml:"dbdsn"`
}

func LoadConfigFromFile(filename string) (Config, error) {
	var yamlConfig Config
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return yamlConfig, err
	}
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return yamlConfig, err
	}
	return yamlConfig, err
}

func (cfg *Config) IsValid() bool {
	// Check DbDSN is set
	if cfg.DbDSN == "" {
		return false
	}
	return true
}
