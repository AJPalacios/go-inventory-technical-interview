package config

import (
	"fmt"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig("../..")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("ServerPort: '%s'\n", cfg.ServerPort)
	fmt.Printf("DBPath: '%s'\n", cfg.DBPath)
	fmt.Printf("Environment: '%s'\n", cfg.Environment)
	fmt.Printf("LogLevel: '%s'\n", cfg.LogLevel)

	if cfg.ServerPort == "" {
		t.Error("ServerPort is empty")
	}
	if cfg.DBPath == "" {
		t.Error("DBPath is empty")
	}
	if cfg.Environment == "" {
		t.Error("Environment is empty")
	}
	if cfg.LogLevel == "" {
		t.Error("LogLevel is empty")
	}
}
