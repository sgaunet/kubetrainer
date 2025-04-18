package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type RedisConfig struct {
	RedisDSN         string `yaml:"dsn"                env:"DSN"`
	MaxStreamLength  int    `yaml:"maxstreamlength"    env:"MAXSTREAMLENGTH"`
	RedisStreamName  string `yaml:"streamname"         env:"STREAMNAME"`
	RedisStreamGroup string `yaml:"streamgroup"        env:"STREAMGROUP"`
}

type DBConfig struct {
	DbDSN string `yaml:"dsn" env:"DSN"`
}

// ConfigRepository is a struct that holds the configuration for the repository (database)
type Config struct {
	DBCfg    DBConfig    `yaml:"db"    envPrefix:"DB_"`
	RedisCfg RedisConfig `yaml:"redis" envPrefix:"REDIS_"`
}

func LoadConfigFromFile(filename string) (*Config, error) {
	var yamlConfig Config
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML file: %w", err)
	}
	return &yamlConfig, err
}

func (cfg *Config) IsDBConfig() bool {
	// Check DbDSN is set
	if cfg.DBCfg.DbDSN == "" {
		return false
	}
	return true
}

func (cfg *Config) IsRedisConfig() bool {
	// Check RedisDSN is set
	if cfg.RedisCfg.RedisDSN == "" {
		return false
	}
	if cfg.RedisCfg.MaxStreamLength == 0 {
		return false
	}
	if cfg.RedisCfg.RedisStreamName == "" {
		return false
	}
	if cfg.RedisCfg.RedisStreamGroup == "" {
		return false
	}
	return true
}
