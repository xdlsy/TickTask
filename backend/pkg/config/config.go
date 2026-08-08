package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	CORS    CORSConfig    `yaml:"cors"`
	AI      AIConfig      `yaml:"ai"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type AIConfig struct {
	Provider string        `yaml:"provider"`
	BaseURL  string        `yaml:"base_url"`
	Model    string        `yaml:"model"`
	Timeout  time.Duration `yaml:"timeout"`
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadDefault 加载默认配置
func LoadDefault() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "release",
		},
		Database: DatabaseConfig{
			Path: "./data/ticktask.db",
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"http://localhost:5173"},
		},
		AI: AIConfig{
			Provider: "openai",
			BaseURL:  "https://api.openai.com/v1",
			Model:    "gpt-4o-mini",
			Timeout:  30 * time.Second,
		},
	}
}
