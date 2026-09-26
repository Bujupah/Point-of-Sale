// Package config loads the small on-disk config.json (brief §91). Business
// configuration lives in SQLite settings, not here — this file only says
// where things are and which port to bind.
package config

import (
	"encoding/json"
	"os"
)

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DatabaseConfig struct {
	Path string `json:"path"`
}

type LoggingConfig struct {
	Path string `json:"path"`
}

type Config struct {
	Server     ServerConfig   `json:"server"`
	Database   DatabaseConfig `json:"database"`
	Logging    LoggingConfig  `json:"logging"`
	Migrations string         `json:"migrations_path"`
	Frontend   string         `json:"frontend_path"`
	Assets     string         `json:"assets_path"`
}

func Default() Config {
	return Config{
		Server:     ServerConfig{Host: "127.0.0.1", Port: 17831},
		Database:   DatabaseConfig{Path: "./data/pos.db"},
		Logging:    LoggingConfig{Path: "./logs/pos.log"},
		Migrations: "./migrations",
		Frontend:   "./frontend/dist",
		Assets:     "./assets",
	}
}

// Load reads config.json at path if present, falling back to defaults for
// any field it doesn't set. A missing file is not an error — first run on a
// freshly installed machine has none yet.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
