package mcphost

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"smart-spotlight-wails/backend/packages/llm/history"
	"smart-spotlight-wails/backend/packages/llm/models"
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

func createProvider(ctx context.Context, providerName, modelName, systemPrompt string) (models.Provider, error) {
	switch providerName {
	case "openai":
		apiKey := openaiAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}

		if apiKey == "" {
			return nil, fmt.Errorf(
				"OpenAI API key not provided. Use --openai-api-key flag or OPENAI_API_KEY environment variable",
			)
		}
		return openai.NewProvider(apiKey, openaiBaseURL, modelName, systemPrompt), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// MCPSettings represents the MCP configuration settings
type MCPSettings struct {
	ConfigFile       string
	SystemPromptFile string
	MessageWindow    int
	ProviderName     string // Changed from Model to separate provider and model
	ModelName        string // Added separate model name
	OpenAIBaseURL    string
	AnthropicBaseURL string
	OpenAIAPIKey     string
	AnthropicAPIKey  string
	GoogleAPIKey     string
	DebugMode        bool
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
		settings = &MCPSettings{
			MessageWindow: 10,
			ProviderName:  "openai", // Default provider
			ModelName:     "gpt-4",  // Default model
		}
	}

	logLevel := slog.LevelInfo
	if settings.DebugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	systemPrompt := ""

	provider, err := createProvider(context.Background(), settings.ProviderName, settings.ModelName, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("error creating provider: %w", err)
	}

	return &MCPService{
		settings:       settings,
		provider:       provider,
		logger:         logger,
		initialBackoff: 1 * time.Second,
		maxBackoff:     30 * time.Second,
		maxRetries:     5,
	}, nil
}

// Search performs a search using the configured MCP provider
func (s *MCPService) Search(query string) (*models.MCPResponse, error) {
	ctx := context.Background()
	messages := make([]history.HistoryMessage, 0)

	// Initialize MCP clients if not already done
	if s.mcpClients == nil {
		mcpConfig, err := loadMCPConfig()
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

	// Handle tool calls if any
	for _, toolCall := range message.GetToolCalls() {
		s.logger.Info("using tool", "name", toolCall.GetName())

		parts := strings.Split(toolCall.GetName(), "__")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tool name format: %s", toolCall.GetName())
		}

		serverName, toolName := parts[0], parts[1]
		mcpClient, ok := s.mcpClients[serverName]
		if !ok {
			return nil, fmt.Errorf("server not found: %s", serverName)
		}

		req := mcp.CallToolRequest{}
		req.Params.Name = toolName
		req.Params.Arguments = toolCall.GetArguments()

		toolResult, err := mcpClient.CallTool(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("error calling tool %s: %w", toolName, err)
		}

		if toolResult.Content != nil {
			var resultText string
			for _, item := range toolResult.Content {
				if contentMap, ok := item.(mcp.TextContent); ok {
					resultText += fmt.Sprintf("%v ", contentMap.Text)
				}
			}

			// Create tool result block
			toolResultBlock := history.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolCall.GetID(),
				Content:   toolResult.Content,
				Text:      strings.TrimSpace(resultText),
			}

			// Add tool result to messages
			messages = append(messages, history.HistoryMessage{
				Role:    "tool",
				Content: []history.ContentBlock{toolResultBlock},
			})

			// Convert updated messages for another LLM call
			llmMessages = make([]models.Message, len(messages))
			for i := range messages {
				llmMessages[i] = &messages[i]
			}

			// Get LLM's response to tool results
			message, err = s.provider.CreateMessage(ctx, "", llmMessages, s.tools)
			if err != nil {
				return nil, fmt.Errorf("error creating follow-up message: %w", err)
			}

			// Update response with tool result and LLM's interpretation
			response.Content = message.GetContent()
		}
	}

	return response, nil
}
