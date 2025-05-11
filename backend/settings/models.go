package settings

import "os"

// Settings represents application settings
type Settings struct {
	BaseURL         string   `json:"baseUrl"`
	APIKey          string   `json:"apiKey"`
	Model           string   `json:"model"`
	AvailableModels []string `json:"availableModels"`
}

// DefaultSettings returns default application settings
func DefaultSettings() *Settings {
	return &Settings{
		BaseURL:         os.Getenv("SPOT_AI_API_ENDPOINT"),
		APIKey:          os.Getenv("SPOT_AI_API_KEY"),
		Model:           os.Getenv("SPOT_AI_MODEL"),
		AvailableModels: DefaultModels(),
	}
}

// DefaultModels returns the list of available models
func DefaultModels() []string {
	return []string{
		"google/gemini-2.5-pro-exp-03-25",
		"gpt-4",
		"gpt-3.5-turbo",
		"anthropic/claude-3-opus",
		"anthropic/claude-3-sonnet",
		"anthropic/claude-2",
	}
}
