package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Gateway  GatewayConfig  `yaml:"gateway"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type GatewayConfig struct {
	MaxBodySize       int64  `yaml:"max_body_size"`
	MaxAccountRetries int    `yaml:"max_account_retries"`
	PricingURL        string `yaml:"pricing_url"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8787,
			Host: "127.0.0.1",
		},
		Gateway: GatewayConfig{
			MaxBodySize:       50 * 1024 * 1024,
			MaxAccountRetries: 3,
		},
		Database: DatabaseConfig{
			Path: "data.db",
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

func Load(baseDir string) (*Config, error) {
	path := filepath.Join(baseDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()
			if err := cfg.Save(baseDir); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Save(baseDir string) error {
	path := filepath.Join(baseDir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
