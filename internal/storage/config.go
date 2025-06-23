package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DatabasePath   string `yaml:"database_path" json:"database_path"`
	EnableWALMode  bool   `yaml:"enable_wal_mode" json:"enable_wal_mode"`
	MaxConnections int    `yaml:"max_connections" json:"max_connections"`
	BackupEnabled  bool   `yaml:"backup_enabled" json:"backup_enabled"`
	BackupPath     string `yaml:"backup_path" json:"backup_path"`
}

func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configDir := filepath.Join(homeDir, ".config", "ccmgr-ultra")

	return &Config{
		DatabasePath:   filepath.Join(configDir, "data.db"),
		EnableWALMode:  true,
		MaxConnections: 25,
		BackupEnabled:  true,
		BackupPath:     filepath.Join(configDir, "backups"),
	}
}

func (c *Config) Validate() error {
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path cannot be empty")
	}

	if c.MaxConnections <= 0 {
		c.MaxConnections = 25
	}

	return nil
}
