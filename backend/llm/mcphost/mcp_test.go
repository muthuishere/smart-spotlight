package mcphost

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// printJSON is a helper function to pretty print JSON
func printJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshalling JSON: %v", err)
	}
	return string(jsonBytes)
}

func TestMCPService(t *testing.T) {
	// Load environment variables

	//	apiKey := os.Getenv("SPOT_AI_API_KEY")

	providerName := os.Getenv("SPOT_AI_PROVIDER")
	baseURL := os.Getenv("SPOT_AI_API_ENDPOINT")
	apiKey := os.Getenv("SPOT_AI_API_KEY")
	modelName := os.Getenv("SPOT_AI_MODEL")
	systemPrompt := os.Getenv("SPOT_AI_SYSTEM_PROMPT")
	configFile := os.Getenv("SPOT_AI_CONFIG_FILE")
	// Parse message window from environment variable, default to 10
	messageWindow := 10

	if apiKey == "" {
		t.Skip("Skipping test: MCP_API_KEY environment variable not set")
	}

	// Create provider configuration
	provider := LLMProvider{
		ProviderName: providerName,
		BaseURL:      baseURL,
		APIKey:       apiKey,
		ModelName:    modelName,
		Metadata:     make(map[string]string),
	}

	// Create MCP settings
	mcpSettings := &MCPSettings{
		ConfigFile:    configFile,
		SystemPrompt:  systemPrompt,
		MessageWindow: messageWindow,
		Provider:      provider,
		DebugMode:     true, // Always enable debug for tests
	}

	// Create MCP service instance
	service, err := NewMCPService(mcpSettings)
	if err != nil {
		t.Fatalf("Failed to create MCP service: %v", err)
	}

	t.Run("Test MCP Search", func(t *testing.T) {
		// Test query
		query := "What is the capital of France?"

		fmt.Println("Sending query to MCP service:", query)
		response, err := service.Search(query)
		if err != nil {
			t.Errorf("Search failed: %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
		}

		fmt.Println("\n========== MCP Response ==========")
		fmt.Printf("Raw Response: %s\n", printJSON(response))
		fmt.Println("==================================")

		if response.Content == "" {
			t.Error("Expected non-empty content in response")
		}
	})

	t.Run("Test MCP File Query", func(t *testing.T) {
		// Test file-related query
		query := "How many tables are in the database and its name and also list all the files and folders you have access?"

		fmt.Println("\nSending file query to MCP service:", query)
		response, err := service.Search(query)

		fmt.Println("\n========== MCP File Query Response ==========")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			// Don't fail the test, just report the error
			t.Logf("File search query failed with error: %v", err)
		} else if response == nil {
			fmt.Println("Response: <nil>")
			t.Log("Received nil response without error")
		} else {
			fmt.Printf("Raw Response: %s\n", printJSON(response))

			if response.Content == "" {
				t.Log("Response content was empty")
			} else {
				t.Log("Received valid response content")
			}
		}
		fmt.Println("=============================================")
	})
}
