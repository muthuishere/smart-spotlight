package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"smart-spotlight-wails/backend/packages/llm/history"
	"smart-spotlight-wails/backend/packages/llm/models"
	"smart-spotlight-wails/backend/packages/llm/providers/anthropic"
	"smart-spotlight-wails/backend/packages/llm/providers/google"
	"smart-spotlight-wails/backend/packages/llm/providers/ollama"
	"smart-spotlight-wails/backend/packages/llm/providers/openai"

	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	configFile       string
	systemPromptFile string
	messageWindow    int
	modelFlag        string
	openaiBaseURL    string
	anthropicBaseURL string
	openaiAPIKey     string
	anthropicAPIKey  string
	googleAPIKey     string
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	maxRetries     = 5
)

func createProvider(ctx context.Context, settings *MCPSettings) (models.Provider, error) {
	providerName := settings.Provider.ProviderName
	modelName := settings.Provider.ModelName
	systemPrompt := settings.SystemPrompt

	// Use the provider configuration
	switch providerName {
	case "openai":
		apiKey := settings.Provider.APIKey

		if apiKey == "" {
			return nil, fmt.Errorf(
				"OpenAI API key not provided for provider %s", providerName,
			)
		}
		return openai.NewProvider(apiKey, settings.Provider.BaseURL, modelName, systemPrompt), nil

	// You can add more provider types here as needed
	case "anthropic":
		apiKey := anthropicAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}

		if apiKey == "" {
			return nil, fmt.Errorf(
				"Anthropic API key not provided. Use --anthropic-api-key flag or ANTHROPIC_API_KEY environment variable",
			)
		}
		return anthropic.NewProvider(apiKey, settings.Provider.BaseURL, modelName, systemPrompt), nil

	case "ollama":
		return ollama.NewProvider(modelName, systemPrompt)

	case "google":
		apiKey := googleAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			// The project structure is provider specific, but Google calls this GEMINI_API_KEY in e.g. AI Studio. Support both.
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		return google.NewProvider(ctx, apiKey, modelName, systemPrompt)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// LLMProvider represents a single LLM provider configuration
type LLMProvider struct {
	ProviderName string
	BaseURL      string
	APIKey       string
	ModelName    string
	Metadata     map[string]string // Flexible metadata for provider-specific settings
}

// MCPSettings represents the MCP configuration settings
type MCPSettings struct {
	ConfigFile    string
	SystemPrompt  string // Actual system prompt content
	MessageWindow int
	Provider      LLMProvider // Single provider configuration
	DebugMode     bool
}

// MCPService handles MCP operations
type MCPService struct {
	settings       *MCPSettings
	provider       models.Provider
	mcpClients     map[string]mcpclient.MCPClient
	tools          []models.Tool
	logger         *slog.Logger
	initialBackoff time.Duration
	maxBackoff     time.Duration
	maxRetries     int
}

// NewMCPService creates a new MCP service
func NewMCPService(settings *MCPSettings) (*MCPService, error) {
	if settings == nil {
		return nil, fmt.Errorf("settings cannot be nil")
	}

	// Validate required provider fields
	if settings.Provider.ProviderName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	if settings.Provider.BaseURL == "" {
		return nil, fmt.Errorf("provider base URL is required")
	}
	if settings.Provider.APIKey == "" {
		return nil, fmt.Errorf("provider API key is required")
	}
	if settings.Provider.ModelName == "" {
		return nil, fmt.Errorf("provider model name is required")
	}

	// Only metadata is optional, initialize if not provided
	if settings.Provider.Metadata == nil {
		settings.Provider.Metadata = make(map[string]string)
	}

	// Set default message window if not specified
	if settings.MessageWindow <= 0 {
		settings.MessageWindow = 10
	}

	logLevel := slog.LevelInfo
	if settings.DebugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	provider, err := createProvider(context.Background(), settings)
	if err != nil {
		return nil, fmt.Errorf("error creating provider: %w", err)
	}

	return &MCPService{
		settings:       settings,
		provider:       provider,
		logger:         logger,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		maxRetries:     maxRetries,
	}, nil
}

// Search performs a search using the configured MCP provider
func (s *MCPService) Search(query string) (*models.MCPResponse, error) {
	ctx := context.Background()
	messages := make([]history.HistoryMessage, 0)

	// Initialize MCP clients if not already done
	if s.mcpClients == nil {
		mcpConfig, err := loadMCPConfig(s.settings)
		if err != nil {
			return nil, fmt.Errorf("error loading MCP config: %w", err)
		}

		clients, err := createMCPClients(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating MCP clients: %w", err)
		}
		s.mcpClients = clients

		var allTools []models.Tool
		for serverName, mcpClient := range s.mcpClients {
			toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
			if err != nil {
				s.logger.Error("error fetching tools",
					"server", serverName,
					"error", err)
				continue
			}
			serverTools := mcpToolsToAnthropicTools(serverName, toolsResult.Tools)
			allTools = append(allTools, serverTools...)
			s.logger.Info("tools loaded",
				"server", serverName,
				"count", len(toolsResult.Tools))
		}
		s.tools = allTools
	}

	// Add user message to history
	messages = append(messages, history.HistoryMessage{
		Role: "user",
		Content: []history.ContentBlock{{
			Type: "text",
			Text: query,
		}},
	})

	// Convert slice of HistoryMessage to slice of Message interface
	var llmMessages []models.Message
	for i := range messages {
		llmMessages = append(llmMessages, &messages[i])
	}

	// Create message with retries for overloaded scenarios
	var message models.Message
	var err error
	backoff := s.initialBackoff
	retries := 0

	for {
		message, err = s.provider.CreateMessage(ctx, query, llmMessages, s.tools)
		if err != nil {
			// For debugging, print more details about the error
			s.logger.Error("Error creating message",
				"error", err,
				"query", query)

			// Check if it's an overloaded error
			if strings.Contains(err.Error(), "overloaded_error") {
				if retries >= s.maxRetries {
					return nil, fmt.Errorf("service is currently overloaded, please try again later")
				}

				s.logger.Warn("Service is overloaded, backing off...",
					"attempt", retries+1,
					"backoff", backoff.String())

				time.Sleep(backoff)
				backoff *= 2
				if backoff > s.maxBackoff {
					backoff = s.maxBackoff
				}
				retries++
				continue
			}
			// If it's not an overloaded error, return immediately
			return nil, fmt.Errorf("error creating message: %w", err)
		}
		break
	}

	response := &models.MCPResponse{
		Content: message.GetContent(),
	}

	// Store the initial response message in history
	var messageContent []history.ContentBlock
	if message.GetContent() != "" {
		messageContent = append(messageContent, history.ContentBlock{
			Type: "text",
			Text: message.GetContent(),
		})
	} else {
		// Ensure we always have content, even if it's minimal
		// This prevents null content in the message history
		messageContent = append(messageContent, history.ContentBlock{
			Type: "text",
			Text: "I'll help with that.",
		})
	}

	// Collect all tool results before generating a follow-up message
	toolResults := []history.ContentBlock{}
	toolCalls := message.GetToolCalls()

	// If there are no tool calls, we can return the response as is
	if len(toolCalls) == 0 {
		if len(messageContent) > 0 {
			messages = append(messages, history.HistoryMessage{
				Role:    message.GetRole(),
				Content: messageContent,
			})
		}
		return response, nil
	}

	// Process all tool calls and collect their results
	for _, toolCall := range toolCalls {
		s.logger.Info("using tool",
			"name", toolCall.GetName(),
			"arguments", toolCall.GetArguments())

		parts := strings.Split(toolCall.GetName(), "__")
		if len(parts) != 2 {
			s.logger.Error("invalid tool name format", "name", toolCall.GetName())
			continue // Skip this tool call but continue with others
		}

		serverName, toolName := parts[0], parts[1]
		mcpClient, ok := s.mcpClients[serverName]
		if !ok {
			s.logger.Error("server not found", "server", serverName)
			continue // Skip this tool call but continue with others
		}

		req := mcp.CallToolRequest{}
		req.Params.Name = toolName
		req.Params.Arguments = toolCall.GetArguments()

		toolResult, err := mcpClient.CallTool(ctx, req)
		if err != nil {
			s.logger.Error("Tool call failed",
				"tool", toolName,
				"error", err,
				"arguments", toolCall.GetArguments())

			// Instead of failing, add error as a tool result and continue
			errMsg := fmt.Sprintf("Error calling tool %s: %v", toolName, err)
			toolResults = append(toolResults, history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Text:      errMsg,
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": errMsg,
					},
				},
			})
			continue
		}

		// Print the raw tool result for debugging
		resultJSON, _ := json.MarshalIndent(toolResult, "", "  ")
		s.logger.Info("Tool result",
			"tool", toolName,
			"result", string(resultJSON))

		if toolResult.Content != nil {
			// Create the tool result block
			var resultText string
			for _, item := range toolResult.Content {
				if contentMap, ok := item.(mcp.TextContent); ok {
					resultText += fmt.Sprintf("%v ", contentMap.Text)
				}
			}

			resultText = strings.TrimSpace(resultText)
			toolResults = append(toolResults, history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content:   toolResult.Content,
				Text:      resultText,
			})
		}
	}

	// Add the initial message to the history
	messages = append(messages, history.HistoryMessage{
		Role:    message.GetRole(),
		Content: messageContent,
	})

	// Add all tool results to history
	for _, toolResult := range toolResults {
		messages = append(messages, history.HistoryMessage{
			Role:    "tool",
			Content: []history.ContentBlock{toolResult},
		})
	}

	// If we have tool results, make another call to get LLM's response to all tool results
	if len(toolResults) > 0 {
		// Instead of trying complex approaches that are failing, let's create a simple, direct follow-up
		// that just includes the original question and the tool results in a clear format

		// Extract text from all tool results
		var toolResultsText string
		for _, result := range toolResults {
			toolResultsText += result.Text + "\n"
		}

		// Create a completely new conversation with clear context
		directPrompt := fmt.Sprintf("I asked: %s\n\nTool results:\n%s\n\nPlease provide a complete answer based on these results.",
			query, toolResultsText)

		s.logger.Info("Making direct follow-up call",
			"prompt", directPrompt)

		// Create a new, clean history with just this prompt
		directMessages := []history.HistoryMessage{
			{
				Role: "user",
				Content: []history.ContentBlock{{
					Type: "text",
					Text: directPrompt,
				}},
			},
		}

		// Convert to LLM messages
		directLLMMessages := make([]models.Message, len(directMessages))
		for i := range directMessages {
			directLLMMessages[i] = &directMessages[i]
		}

		// Make direct call without tools to avoid any tool-related formatting issues
		directResponse, err := s.provider.CreateMessage(ctx, directPrompt, nil, nil)
		if err != nil {
			s.logger.Error("Direct follow-up failed",
				"error", err)

			// Fall back to showing raw tool results
			response.Content = fmt.Sprintf("Tool results:\n\n%s", strings.TrimSpace(toolResultsText))
		} else {
			// Use the direct response
			s.logger.Info("Direct follow-up succeeded")
			response.Content = directResponse.GetContent()
		}
	}

	return response, nil
}
