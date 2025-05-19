package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"smart-spotlight-ai/backend/settings"
	"time"
)

// ChatResponse represents the response from the LLM
type ChatResponse struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// Service handles LLM operations
type Service struct {
	settings *settings.Settings
}

// NewService creates a new LLM service
func NewService(settings *settings.Settings) *Service {
	return &Service{
		settings: settings,
	}
}

// Search performs a search using the configured LLM
func (s *Service) Search(query string) (*ChatResponse, error) {
	messages := []map[string]string{
		{
			"role":    "system",
			"content": "You are a helpful assistant. Respond in markdown format. Keep responses concise but informative.",
		},
		{
			"role":    "user",
			"content": query,
		},
	}

	reqBody := map[string]interface{}{
		"model":       s.settings.Model,
		"messages":    messages,
		"temperature": 0.7,
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error preparing request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.settings.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.settings.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error connecting to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			if errMsg, ok := errorResponse["error"].(map[string]interface{}); ok {
				return &ChatResponse{Error: fmt.Sprintf("%v", errMsg["message"])}, nil
			}
		}
		return &ChatResponse{Error: fmt.Sprintf("API returned status code %d", resp.StatusCode)}, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content format")
	}

	return &ChatResponse{Content: content}, nil
}

// TestAPIConnection tests if the API settings are valid
func (s *Service) TestAPIConnection() error {
	// Create a simple request to test the API
	reqBody := map[string]interface{}{
		"model": s.settings.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Hello, this is a test message.",
			},
		},
		"max_tokens":  5,   // Fixed small value for testing
		"temperature": 0.7, // Fixed default value
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error preparing request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", s.settings.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.settings.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error connecting to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			if errMsg, ok := errorResponse["error"].(map[string]interface{}); ok {
				return fmt.Errorf("API error: %v", errMsg["message"])
			}
		}
		return fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	return nil
}
