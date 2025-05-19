package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"smart-spotlight-ai/backend/packages/llm/models"

	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	transportStdio = "stdio"
	transportSSE   = "sse"
)

type MCPConfig struct {
	MCPServers map[string]ServerConfigWrapper `json:"mcpServers"`
}

type ServerConfig interface {
	GetType() string
}

type STDIOServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

func (s STDIOServerConfig) GetType() string {
	return transportStdio
}

type SSEServerConfig struct {
	Url     string   `json:"url"`
	Headers []string `json:"headers,omitempty"`
}

func (s SSEServerConfig) GetType() string {
	return transportSSE
}

type ServerConfigWrapper struct {
	Config ServerConfig
}

func (w *ServerConfigWrapper) UnmarshalJSON(data []byte) error {
	var typeField struct {
		Url string `json:"url"`
	}

	if err := json.Unmarshal(data, &typeField); err != nil {
		return err
	}
	if typeField.Url != "" {
		var sse SSEServerConfig
		if err := json.Unmarshal(data, &sse); err != nil {
			return err
		}
		w.Config = sse
	} else {
		var stdio STDIOServerConfig
		if err := json.Unmarshal(data, &stdio); err != nil {
			return err
		}
		w.Config = stdio
	}

	return nil
}

func (w ServerConfigWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.Config)
}

func mcpToolsToAnthropicTools(serverName string, mcpTools []mcp.Tool) []models.Tool {
	anthropicTools := make([]models.Tool, len(mcpTools))

	for i, tool := range mcpTools {
		namespacedName := fmt.Sprintf("%s__%s", serverName, tool.Name)

		anthropicTools[i] = models.Tool{
			Name:        namespacedName,
			Description: tool.Description,
			InputSchema: models.Schema{
				Type:       tool.InputSchema.Type,
				Properties: tool.InputSchema.Properties,
				Required:   tool.InputSchema.Required,
			},
		}
	}

	return anthropicTools
}

func loadMCPConfig(settings *MCPSettings) (*MCPConfig, error) {
	var configPath string
	if settings.ConfigFile != "" {
		configPath = settings.ConfigFile
	} else {
		return nil, fmt.Errorf("config file not specified")
	}

	fmt.Println("Loading MCP config from:", configPath)
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config without creating a file
		defaultConfig := MCPConfig{
			MCPServers: make(map[string]ServerConfigWrapper),
		}
		return &defaultConfig, nil
	}

	// Read existing config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", configPath, err)
	}

	var config MCPConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

func createMCPClients(
	config *MCPConfig,
) (map[string]mcpclient.MCPClient, error) {
	clients := make(map[string]mcpclient.MCPClient)

	for name, server := range config.MCPServers {
		var client mcpclient.MCPClient
		var err error

		if server.Config.GetType() == transportSSE {
			sseConfig := server.Config.(SSEServerConfig)

			options := []mcpclient.ClientOption{}

			if sseConfig.Headers != nil {
				// Parse headers from the config
				headers := make(map[string]string)
				for _, header := range sseConfig.Headers {
					parts := strings.SplitN(header, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						headers[key] = value
					}
				}
				options = append(options, mcpclient.WithHeaders(headers))
			}

			client, err = mcpclient.NewSSEMCPClient(
				sseConfig.Url,
				options...,
			)
			if err == nil {
				err = client.(*mcpclient.SSEMCPClient).Start(context.Background())
			}
		} else {
			stdioConfig := server.Config.(STDIOServerConfig)
			var env []string
			for k, v := range stdioConfig.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			client, err = mcpclient.NewStdioMCPClient(
				stdioConfig.Command,
				env,
				stdioConfig.Args...)
		}
		if err != nil {
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf(
				"failed to create MCP client for %s: %w",
				name,
				err,
			)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		slog.Info("Initializing server...", "name", name)
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "mcphost",
			Version: "0.1.0",
		}
		initRequest.Params.Capabilities = mcp.ClientCapabilities{}

		_, err = client.Initialize(ctx, initRequest)
		if err != nil {
			client.Close()
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf(
				"failed to initialize MCP client for %s: %w",
				name,
				err,
			)
		}

		clients[name] = client
	}

	return clients, nil
}

// ServerConfigService handles MCP server configuration operations
type ServerConfigService struct {
	settings *MCPSettings
	clients  map[string]mcpclient.MCPClient
	logger   *slog.Logger
}

// NewServerConfigService creates a new server configuration service
func NewServerConfigService(settings *MCPSettings) *ServerConfigService {
	return &ServerConfigService{
		settings: settings,
		clients:  make(map[string]mcpclient.MCPClient),
		logger:   slog.Default(),
	}
}

// LoadConfig loads the MCP configuration and initializes clients
func (s *ServerConfigService) LoadConfig() error {
	config, err := loadMCPConfig(s.settings)
	if err != nil {
		return fmt.Errorf("error loading MCP config: %w", err)
	}

	clients, err := createMCPClients(config)
	if err != nil {
		return fmt.Errorf("error creating MCP clients: %w", err)
	}

	s.clients = clients
	return nil
}

// GetClients returns the initialized MCP clients
func (s *ServerConfigService) GetClients() map[string]mcpclient.MCPClient {
	return s.clients
}

// CloseClients closes all MCP client connections
func (s *ServerConfigService) CloseClients() {
	for name, client := range s.clients {
		if err := client.Close(); err != nil {
			s.logger.Error("failed to close server",
				"name", name,
				"error", err)
		} else {
			s.logger.Info("server closed", "name", name)
		}
	}
}

// ListTools retrieves all available tools from all MCP clients
func (s *ServerConfigService) ListTools(ctx context.Context) ([]models.Tool, error) {
	var allTools []models.Tool

	for serverName, mcpClient := range s.clients {
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

	return allTools, nil
}
