package mcphost

import (
	"context"
	"log/slog"
	"regexp"
	"smart-spotlight-ai/backend/packages/llm/models"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
)

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

type PromptEvent struct {
	Type string      // see consts above
	Data interface{} // string, HistoryMessage, map[string]any, etc.
}

type confirmationReply struct {
	Token string
	OK    bool
}

// MCPService handles MCP operations
type MCPService struct {
	ctx context.Context // ← add

	settings       *MCPSettings
	provider       models.Provider
	mcpClients     map[string]mcpclient.MCPClient
	tools          []models.Tool
	logger         *slog.Logger
	initialBackoff time.Duration
	maxBackoff     time.Duration
	maxRetries     int
	waitingConfirm bool
	InputChan      chan PromptEvent       // receive prompts / confirmations
	EventChan      chan PromptEvent       // emit tool_use / final_result / …
	ConfirmChan    chan confirmationReply // inside struct

}

const (
	EventPrompt               = "prompt"
	EventToolUse              = "tool_use"
	EventToolResult           = "tool_result"
	EventAuthorization        = "authorization_required"
	EventConfirmationRequired = "confirmation_required"
	EventFinalResult          = "final_result"
	EventError                = "error"
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

var mutatingVerbs = []string{
	"create", "insert", "add",
	"update", "modify", "patch", "put",
	"delete", "remove", "drop",
	"write", "send", "post", "publish",
}

// build regex once at init
var writeActionRe = regexp.MustCompile(`(?i)\b(` + strings.Join(mutatingVerbs, "|") + `)\b`)
