package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ServerConfig struct {
	LogLevel                      LogLevel      `json:"logLevel"`
	ShowTimestamps                bool          `json:"showTimestamps"`
	MaxRequestActive              int           `json:"maxRequestActive"`
	LogFile                       string        `json:"logFile"`
	OutgoingMessageTimeoutSeconds time.Duration `json:"outgoingMessageTimeoutSeconds"`
}

func NewDefaultConfig() ServerConfig {
	return ServerConfig{
		LogLevel:                      LogInfo,
		ShowTimestamps:                true,
		MaxRequestActive:              10,
		LogFile:                       "",
		OutgoingMessageTimeoutSeconds: 15, // Increased from 5 to 15 seconds
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
