package llm

import (
	"fmt"
	"os"
	"smart-spotlight-ai/backend/settings"
	"testing"
)

func TestLLMService(t *testing.T) {
	// Load environment variables
	apiKey := os.Getenv("SPOT_AI_API_KEY")
	model := os.Getenv("SPOT_AI_MODEL")
	baseURL := os.Getenv("SPOT_AI_API_ENDPOINT")

	if apiKey == "" || model == "" || baseURL == "" {
		t.Skip("Skipping test: Required environment variables not set")
	}

	// Create settings for test
	testSettings := &settings.Settings{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
	}

	// Create service instance
	service := NewService(testSettings)

	t.Run("Test LLM Search", func(t *testing.T) {
		response, err := service.Search("What is clojure?")
		if err != nil {
			t.Errorf("Search failed: %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
		}
		fmt.Printf("Response: %+v\n", response)
		if response.Content == "" {
			t.Error("Expected non-empty content in response")
		}
		if response.Error != "" {
			t.Errorf("Unexpected error in response: %s", response.Error)
		}
	})
}
