package internal

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigManager handles loading and saving configuration.
type ConfigManager struct {
	Config *Config
}

// NewConfigManager creates a new ConfigManager.
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		Config: &Config{},
	}
}

// LoadConfig loads the configuration from the config.yaml file.
func (cm *ConfigManager) LoadConfig() error {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, cm.Config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// SaveConfig saves the configuration to the config.yaml file.
func (cm *ConfigManager) SaveConfig() error {
	data, err := yaml.Marshal(cm.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile("config.yaml", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}
