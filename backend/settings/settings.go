package settings

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

// AppSettings holds the current application settings instance
var AppSettings *Settings

// InitSettings initializes the application settings
func InitSettings() {
	var err error
	AppSettings, err = LoadSettings()
	if err != nil {
		log.Printf("Error loading settings: %v. Using default settings.", err)
		AppSettings = DefaultSettings()

		slog.Info("Settings initialized with default values", "settings", AppSettings)
		// Try to save the default settings
		if saveErr := SaveSettings(AppSettings); saveErr != nil {
			log.Printf("Error saving initial default settings: %v", saveErr)
		}
	}
}

// GetCurrentSettings returns the current settings instance
func GetCurrentSettings() *Settings {
	if AppSettings == nil {
		InitSettings()
	}
	return AppSettings
}

// UpdateSettings updates the current settings and saves them to disk
func UpdateSettings(newSettings *Settings) error {
	if err := SaveSettings(newSettings); err != nil {
		return err
	}
	AppSettings = newSettings
	return nil
}

func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "smart-spotlight-wails")
	if err := os.MkdirAll(appConfigDir, 0750); err != nil {
		return "", err
	}
	return appConfigDir, nil
}

const settingsFile = "settings.json"

// SaveSettings saves the settings to a file
func SaveSettings(s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	appConfigDir := filepath.Join(configDir, "smart-spotlight-wails")
	if err := os.MkdirAll(appConfigDir, 0750); err != nil {
		return err
	}
	filePath := filepath.Join(appConfigDir, settingsFile)
	return os.WriteFile(filePath, data, 0600)
}

// LoadSettings loads the settings from a file
func LoadSettings() (*Settings, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appConfigDir := filepath.Join(configDir, "smart-spotlight-wails")
	filePath := filepath.Join(appConfigDir, settingsFile)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default settings if file doesn't exist
			return DefaultSettings(), nil
		}
		return nil, err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	// Ensure available models are always present
	if len(s.AvailableModels) == 0 {
		s.AvailableModels = DefaultModels()
	}

	return &s, nil
}
