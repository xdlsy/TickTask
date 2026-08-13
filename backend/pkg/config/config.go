package config

import (
	"os"
	"path/filepath"
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

// AppDir 返回用户级 TickTask 根目录（Windows 为 %APPDATA%\TickTask）。
// 操作系统用户配置目录不可用时 ok=false。
func AppDir() (string, bool) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return "", false
	}
	return filepath.Join(base, "TickTask"), true
}

// Resolve 加载生效配置：CWD 的 configs/config.yaml 优先（仓库开发布局），
// 其次 <AppDir>/config.yaml（打包 exe），都没有则用默认值，并把数据库
// 落到 <AppDir>/data/ticktask.db。第二个返回值为配置来源路径（默认值时为 ""）。
func Resolve() (*Config, string) {
	candidates := []string{filepath.Join("configs", "config.yaml")}
	if appDir, ok := AppDir(); ok {
		candidates = append(candidates, filepath.Join(appDir, "config.yaml"))
	}
	for _, p := range candidates {
		if cfg, err := Load(p); err == nil {
			return cfg, p
		}
	}
	cfg := LoadDefault()
	if appDir, ok := AppDir(); ok {
		cfg.Database.Path = filepath.Join(appDir, "data", "ticktask.db")
	}
	return cfg, ""
}
