package server

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	LogLevel         LogLevel `json:"logLevel"`
	ShowTimestamps   bool     `json:"showTimestamps"`
	MaxRequestActive int      `json:"maxRequestActive"`
	LogFile          string   `json:"logFile"`
}

func NewDefaultConfig() ServerConfig {
	return ServerConfig{
		LogLevel:         LogInfo,
		ShowTimestamps:   true,
		MaxRequestActive: 10,
		LogFile:          "",
	}
}

func LoadConfigFromFile(path string) (ServerConfig, error) {
	config := NewDefaultConfig()

	if path == "" {
		return config, nil
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	err = json.Unmarshal(fileData, &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
