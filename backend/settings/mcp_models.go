// filepath: /Users/muthuishere/muthu/gitworkspace/mcp/smar-spotlight-mcp-host/backend/settings/mcp_models.go
package settings

import (
	"encoding/json"
)

const (
	transportStdio = "stdio"
	transportSSE   = "sse"
)

// MCPServerConfig represents the configuration structure for all MCP servers
type MCPServerConfig struct {
	MCPServers map[string]ServerConfigWrapper `json:"mcpServers"`
}

// ServerConfig is an interface that all server types must implement
type ServerConfig interface {
	GetType() string
}

// ServerConfigWrapper wraps different types of server configurations
type ServerConfigWrapper struct {
	Config ServerConfig
	// Enabled indicates if this server should be used
	Enabled bool `json:"enabled"`
}

// UnmarshalJSON custom unmarshaler for ServerConfigWrapper
func (w *ServerConfigWrapper) UnmarshalJSON(data []byte) error {
	var objMap map[string]interface{}
	if err := json.Unmarshal(data, &objMap); err != nil {
		return err
	}

	// Check for enabled field (default to true if not present)
	if enabled, ok := objMap["enabled"]; ok {
		if enabledBool, ok := enabled.(bool); ok {
			w.Enabled = enabledBool
		} else {
			w.Enabled = true
		}
	} else {
		w.Enabled = true
	}

	// Check for URL to determine if it's SSE
	if _, hasURL := objMap["url"]; hasURL {
		var sseConfig SSEServerConfig
		if err := json.Unmarshal(data, &sseConfig); err != nil {
			return err
		}
		w.Config = sseConfig
		return nil
	}

	// Otherwise, assume it's STDIO
	var stdioConfig STDIOServerConfig
	if err := json.Unmarshal(data, &stdioConfig); err != nil {
		return err
	}
	w.Config = stdioConfig
	return nil
}

// MarshalJSON custom marshaler for ServerConfigWrapper
func (w ServerConfigWrapper) MarshalJSON() ([]byte, error) {
	var result map[string]interface{}

	// Convert the Config to a map
	configBytes, err := json.Marshal(w.Config)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configBytes, &result); err != nil {
		return nil, err
	}

	// Add enabled field
	result["enabled"] = w.Enabled

	return json.Marshal(result)
}

// STDIOServerConfig represents configuration for a command-line based MCP server
type STDIOServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// GetType returns the type of this server config
func (s STDIOServerConfig) GetType() string {
	return transportStdio
}

// SSEServerConfig represents configuration for a web-based MCP server
type SSEServerConfig struct {
	Url     string   `json:"url"`
	Headers []string `json:"headers,omitempty"`
}

// GetType returns the type of this server config
func (s SSEServerConfig) GetType() string {
	return transportSSE
}

// ActiveMCPServers represents a list of server names that are currently active
type ActiveMCPServers struct {
	ActiveServers []string `json:"activeServers"`
}

// NewMCPServerConfig creates a new empty MCP server configuration
func NewMCPServerConfig() *MCPServerConfig {
	return &MCPServerConfig{
		MCPServers: make(map[string]ServerConfigWrapper),
	}
}

// NewActiveMCPServers creates a new empty active MCP servers list
func NewActiveMCPServers() *ActiveMCPServers {
	return &ActiveMCPServers{
		ActiveServers: []string{},
	}
}
