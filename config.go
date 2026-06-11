package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Printer string `json:"printer"`
}

func configPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

func loadConfig() Config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return Config{}
	}
	var c Config
	json.Unmarshal(data, &c)
	return c
}

func saveConfig(c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
