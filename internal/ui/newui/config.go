package newui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
)

// AppConfig stores persistent application settings
type AppConfig struct {
	BoardColorTheme int `json:"board_color_theme"` // Store as int for JSON compatibility
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	var configDir string
	// Use platform-appropriate config directory
	if os.Getenv("APPDATA") != "" {
		// Windows: use %APPDATA%\OpenTraceJTAG
		configDir = filepath.Join(os.Getenv("APPDATA"), "OpenTraceJTAG")
	} else {
		// Linux/macOS: use ~/.config/opentracejtag
		configDir = filepath.Join(homeDir, ".config", "opentracejtag")
	}
	
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	
	return filepath.Join(configDir, "config.json"), nil
}

// LoadConfig loads the application configuration
func LoadConfig() (*AppConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return &AppConfig{}, err
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &AppConfig{
				BoardColorTheme: int(renderer.ThemeClassic),
			}, nil
		}
		return nil, err
	}
	
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SaveConfig saves the application configuration
func SaveConfig(config *AppConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}
