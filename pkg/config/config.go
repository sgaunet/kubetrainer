// Package config provides loading and access for kubetrainer configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v2"
)

const (
	bytesPerKB    = 1024
	defaultDataMB = 1024
	defaultDataGB = 1024
)

// RedisConfig holds Redis connection and stream parameters.
type RedisConfig struct {
	RedisDSN         string `env:"DSN"             yaml:"dsn"`
	MaxStreamLength  int    `env:"MAXSTREAMLENGTH" yaml:"maxstreamlength"`
	RedisStreamName  string `env:"STREAMNAME"      yaml:"streamname"`
	RedisStreamGroup string `env:"STREAMGROUP"     yaml:"streamgroup"`
}

// DBConfig holds the database connection settings.
type DBConfig struct {
	DbDSN string `env:"DSN" yaml:"dsn"`
}

// ProducerConfig holds the producer-side simulation settings.
type ProducerConfig struct {
	DataSizeBytes int64 `env:"DATA_SIZE_BYTES" yaml:"data_size_bytes"`
}

// Config aggregates the configuration for the database, redis, and producer.
type Config struct {
	DBCfg       DBConfig       `envPrefix:"DB_"       yaml:"db"`
	RedisCfg    RedisConfig    `envPrefix:"REDIS_"    yaml:"redis"`
	ProducerCfg ProducerConfig `envPrefix:"PRODUCER_" yaml:"producer"`
}

// LoadConfigFromFile reads and parses a YAML configuration file.
func LoadConfigFromFile(filename string) (*Config, error) {
	var yamlConfig Config
	cleanPath := filepath.Clean(filename)
	yamlFile, err := os.ReadFile(cleanPath) // #nosec G304 -- path provided via CLI flag by operator
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}
	if err := yaml.Unmarshal(yamlFile, &yamlConfig); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML file: %w", err)
	}
	return &yamlConfig, nil
}

// LoadConfigFromEnv returns a new Config struct from the environment variables.
func LoadConfigFromEnv() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing environment variables: %w", err)
	}
	return &cfg, nil
}

// IsDBConfig reports whether the database section is configured.
func (cfg *Config) IsDBConfig() bool {
	return cfg.DBCfg.DbDSN != ""
}

// IsRedisConfig reports whether the redis section is fully configured.
func (cfg *Config) IsRedisConfig() bool {
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

// DefaultDataSize returns the configured data size in bytes, or 1GB if not set.
func (cfg *Config) DefaultDataSize() int64 {
	if cfg.ProducerCfg.DataSizeBytes <= 0 {
		return bytesPerKB * defaultDataMB * defaultDataGB
	}
	return cfg.ProducerCfg.DataSizeBytes
}
