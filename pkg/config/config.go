package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
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
type ProducerConfig struct {
	DataSizeBytes int64 `yaml:"data_size_bytes" env:"DATA_SIZE_BYTES"`
}

type Config struct {
	DBCfg       DBConfig       `yaml:"db"       envPrefix:"DB_"`
	RedisCfg    RedisConfig    `yaml:"redis"    envPrefix:"REDIS_"`
	ProducerCfg ProducerConfig `yaml:"producer" envPrefix:"PRODUCER_"`
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

// LoadConfigFromEnv returns a new Config struct from the environment variables
func LoadConfigFromEnv() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
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

// DefaultDataSize returns the configured data size in bytes, or 1GB if not set
func (cfg *Config) DefaultDataSize() int64 {
	if cfg.ProducerCfg.DataSizeBytes <= 0 {
		return 1024 * 1024 * 1024 // 1GB default
	}
	return cfg.ProducerCfg.DataSizeBytes
}
