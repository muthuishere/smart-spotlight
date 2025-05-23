// filepath: /Users/muthuishere/muthu/gitworkspace/mcp/smar-spotlight-mcp-host/backend/settingsapi.go
package backend

import (
	"fmt"
	"log/slog"
	"smart-spotlight-ai/backend/settings"
)

// MCPServerInfo represents server information exposed to the frontend
type MCPServerInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	IsActive  bool                   `json:"isActive"`
	Config    map[string]interface{} `json:"config"`
}

// GetMCPServers returns all MCP server configurations
func (a *App) GetMCPServers() []MCPServerInfo {
	if a.mcpServerSettingsService == nil {
		slog.Error("MCP server settings service not initialized")
		return []MCPServerInfo{}
	}

	// Get all servers and active server names
	allServers := a.mcpServerSettingsService.GetAllServers()
	activeServers := a.mcpServerSettingsService.GetActiveServerNames()

	// Create a map of active server names for quick lookup
	activeMap := make(map[string]bool)
	for _, name := range activeServers {
		activeMap[name] = true
	}

	// Convert to frontend-friendly format
	result := make([]MCPServerInfo, 0, len(allServers))
	for name, server := range allServers {
		var configMap map[string]interface{}

		// Convert server config to map based on its type
		switch server.Config.GetType() {
		case "stdio":
			if stdioConfig, ok := server.Config.(settings.STDIOServerConfig); ok {
				configMap = map[string]interface{}{
					"command": stdioConfig.Command,
					"args":    stdioConfig.Args,
					"env":     stdioConfig.Env,
				}
			}
		case "sse":
			if sseConfig, ok := server.Config.(settings.SSEServerConfig); ok {
				configMap = map[string]interface{}{
					"url":     sseConfig.Url,
					"headers": sseConfig.Headers,
				}
			}
		default:
			configMap = map[string]interface{}{}
		}

		result = append(result, MCPServerInfo{
			Name:     name,
			Type:     server.Config.GetType(),
			Enabled:  server.Enabled,
			IsActive: activeMap[name],
			Config:   configMap,
		})
	}

	return result
}

// AddMCPSTDIOServer adds a new STDIO-based MCP server
func (a *App) AddMCPSTDIOServer(name, command string, args []string, env map[string]string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.AddSTDIOServer(name, command, args, env)
}

// AddMCPSSEServer adds a new SSE-based MCP server
func (a *App) AddMCPSSEServer(name, url string, headers []string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.AddSSEServer(name, url, headers)
}

// DeleteMCPServer removes an MCP server
func (a *App) DeleteMCPServer(name string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.DeleteServer(name)
}

// EnableMCPServer enables an MCP server
func (a *App) EnableMCPServer(name string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.EnableServer(name)
}

// DisableMCPServer disables an MCP server
func (a *App) DisableMCPServer(name string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.DisableServer(name)
}

// SetMCPServerEnabled sets the enabled state of an MCP server
func (a *App) SetMCPServerEnabled(name string, enabled bool) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	return a.mcpServerSettingsService.SetServerEnabled(name, enabled)
}

// UpdateMCPSTDIOServer updates an existing STDIO server configuration
func (a *App) UpdateMCPSTDIOServer(name, command string, args []string, env map[string]string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	// Create the server configuration wrapper
	serverConfig := settings.ServerConfigWrapper{
		Config: settings.STDIOServerConfig{
			Command: command,
			Args:    args,
			Env:     env,
		},
		Enabled: true, // Default to enabled, can be changed separately
	}

	return a.mcpServerSettingsService.UpdateServer(name, serverConfig)
}

// UpdateMCPSSEServer updates an existing SSE server configuration
func (a *App) UpdateMCPSSEServer(name, url string, headers []string) error {
	if a.mcpServerSettingsService == nil {
		return fmt.Errorf("MCP server settings service not initialized")
	}

	// Create the server configuration wrapper
	serverConfig := settings.ServerConfigWrapper{
		Config: settings.SSEServerConfig{
			Url:     url,
			Headers: headers,
		},
		Enabled: true, // Default to enabled, can be changed separately
	}

	return a.mcpServerSettingsService.UpdateServer(name, serverConfig)
}

// GetMCPConfigPath returns the path to the MCP configuration file
func (a *App) GetMCPConfigPath() string {
	if a.mcpServerSettingsService == nil {
		return ""
	}
	return a.mcpServerSettingsService.GetServersFilePath()
}

// GetActiveMCPConfigPath returns the path to the active MCP servers file
func (a *App) GetActiveMCPConfigPath() string {
	if a.mcpServerSettingsService == nil {
		return ""
	}
	return a.mcpServerSettingsService.GetActiveServersFilePath()
}