// filepath: /Users/muthuishere/muthu/gitworkspace/mcp/smar-spotlight-mcp-host/backend/settings/mcp_settings_service.go
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const (
	mcpServersFileName       = "mcp-servers.json"
	activeMCPServersFileName = "active-mcp-servers.json"
)

// MCPServerSettingsService handles operations for MCP server configurations
type MCPServerSettingsService struct {
	configDir     string
	mutex         sync.RWMutex
	serverConfig  *MCPServerConfig
	activeServers *ActiveMCPServers
	serversFile   string
	activeFile    string
	configLoaded  bool
	activeLoaded  bool
}

// NewMCPServerSettingsService creates a new instance of the MCP server settings service
// If configDir is empty, it will use the default app config directory
func NewMCPServerSettingsService(configDir string) (*MCPServerSettingsService, error) {
	

	// If configDir is empty, use the default app config directory
	if configDir == "" {
		
			return nil, fmt.Errorf("failed to get config directory" )
		
	}

	serversFile := filepath.Join(configDir, mcpServersFileName)
	activeFile := filepath.Join(configDir, activeMCPServersFileName)

	svc := &MCPServerSettingsService{
		configDir:     configDir,
		serversFile:   serversFile,
		activeFile:    activeFile,
		serverConfig:  NewMCPServerConfig(),
		activeServers: NewActiveMCPServers(),
		configLoaded:  false,
		activeLoaded:  false,
	}

	// Load existing configurations if they exist
	if err := svc.LoadConfigurations(); err != nil {
		slog.Warn("Failed to load existing MCP server configurations", "error", err)
	}

	return svc, nil
}

// LoadConfigurations loads both the server configurations and active servers list
func (s *MCPServerSettingsService) LoadConfigurations() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Load server configurations
	if err := s.loadServerConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load server configurations: %w", err)
	}

	// Load active servers list
	if err := s.loadActiveServers(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load active servers list: %w", err)
	}

	return nil
}

// loadServerConfig loads the MCP server configurations from file
func (s *MCPServerSettingsService) loadServerConfig() error {
	data, err := os.ReadFile(s.serversFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.serverConfig = NewMCPServerConfig()
			s.configLoaded = true
			return os.ErrNotExist
		}
		return err
	}

	var config MCPServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal server config: %w", err)
	}

	s.serverConfig = &config
	s.configLoaded = true
	return nil
}

// loadActiveServers loads the list of active MCP servers
func (s *MCPServerSettingsService) loadActiveServers() error {
	data, err := os.ReadFile(s.activeFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.activeServers = NewActiveMCPServers()
			s.activeLoaded = true
			return os.ErrNotExist
		}
		return err
	}

	var active ActiveMCPServers
	if err := json.Unmarshal(data, &active); err != nil {
		return fmt.Errorf("failed to unmarshal active servers: %w", err)
	}

	s.activeServers = &active
	s.activeLoaded = true
	return nil
}

// Save persists both server configurations and active servers list to disk
func (s *MCPServerSettingsService) Save() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if err := s.saveServerConfig(); err != nil {
		return fmt.Errorf("failed to save server configurations: %w", err)
	}

	if err := s.saveActiveServers(); err != nil {
		return fmt.Errorf("failed to save active servers list: %w", err)
	}

	return nil
}

// saveServerConfig saves the MCP server configurations to file
func (s *MCPServerSettingsService) saveServerConfig() error {
	data, err := json.MarshalIndent(s.serverConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	if err := os.WriteFile(s.serversFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}

	return nil
}

// saveActiveServers saves the list of active MCP servers
func (s *MCPServerSettingsService) saveActiveServers() error {
	data, err := json.MarshalIndent(s.activeServers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal active servers: %w", err)
	}

	if err := os.WriteFile(s.activeFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write active servers file: %w", err)
	}

	return nil
}

// GetServersFilePath returns the path to the MCP servers configuration file
func (s *MCPServerSettingsService) GetServersFilePath() string {
	return s.serversFile
}

// GetActiveServersFilePath returns the path to the active MCP servers file
func (s *MCPServerSettingsService) GetActiveServersFilePath() string {
	return s.activeFile
}

// GetAllServers returns all configured MCP servers
func (s *MCPServerSettingsService) GetAllServers() map[string]ServerConfigWrapper {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			slog.Error("Failed to load server config", "error", err)
		}
	}

	// Create a copy to avoid exposing internal state
	result := make(map[string]ServerConfigWrapper)
	for name, config := range s.serverConfig.MCPServers {
		result[name] = config
	}

	return result
}

// GetActiveServerNames returns a list of active server names
func (s *MCPServerSettingsService) GetActiveServerNames() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.activeLoaded {
		if err := s.loadActiveServers(); err != nil {
			slog.Error("Failed to load active servers", "error", err)
		}
	}

	// Create a copy to avoid exposing internal state
	result := make([]string, len(s.activeServers.ActiveServers))
	copy(result, s.activeServers.ActiveServers)

	return result
}

// GetActiveServers returns only the active server configurations
func (s *MCPServerSettingsService) GetActiveServers() map[string]ServerConfigWrapper {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			slog.Error("Failed to load server config", "error", err)
		}
	}

	if !s.activeLoaded {
		if err := s.loadActiveServers(); err != nil {
			slog.Error("Failed to load active servers", "error", err)
		}
	}

	activeServers := make(map[string]ServerConfigWrapper)
	for _, name := range s.activeServers.ActiveServers {
		if server, exists := s.serverConfig.MCPServers[name]; exists {
			activeServers[name] = server
		}
	}

	return activeServers
}

// GetServer returns a specific server configuration by name
func (s *MCPServerSettingsService) GetServer(name string) (ServerConfigWrapper, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			slog.Error("Failed to load server config", "error", err)
		}
	}

	server, exists := s.serverConfig.MCPServers[name]
	return server, exists
}

// AddSTDIOServer adds a new STDIO-based MCP server
func (s *MCPServerSettingsService) AddSTDIOServer(name string, command string, args []string, env map[string]string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	// Check if server already exists
	if _, exists := s.serverConfig.MCPServers[name]; exists {
		return fmt.Errorf("server with name %s already exists", name)
	}

	// Create and add the server
	s.serverConfig.MCPServers[name] = ServerConfigWrapper{
		Config: STDIOServerConfig{
			Command: command,
			Args:    args,
			Env:     env,
		},
		Enabled: true,
	}

	return s.saveServerConfig()
}

// AddSSEServer adds a new SSE-based MCP server
func (s *MCPServerSettingsService) AddSSEServer(name string, url string, headers []string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	// Check if server already exists
	if _, exists := s.serverConfig.MCPServers[name]; exists {
		return fmt.Errorf("server with name %s already exists", name)
	}

	// Create and add the server
	s.serverConfig.MCPServers[name] = ServerConfigWrapper{
		Config: SSEServerConfig{
			Url:     url,
			Headers: headers,
		},
		Enabled: true,
	}

	return s.saveServerConfig()
}

// UpdateServer updates an existing MCP server configuration
func (s *MCPServerSettingsService) UpdateServer(name string, config ServerConfigWrapper) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	// Check if server exists
	if _, exists := s.serverConfig.MCPServers[name]; !exists {
		return fmt.Errorf("server with name %s does not exist", name)
	}

	// Update the server
	s.serverConfig.MCPServers[name] = config

	return s.saveServerConfig()
}

// DeleteServer removes an MCP server configuration
func (s *MCPServerSettingsService) DeleteServer(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	// Check if server exists
	if _, exists := s.serverConfig.MCPServers[name]; !exists {
		return fmt.Errorf("server with name %s does not exist", name)
	}

	// Delete the server
	delete(s.serverConfig.MCPServers, name)

	// Also remove from active servers if present
	if s.activeLoaded {
		for i, activeName := range s.activeServers.ActiveServers {
			if activeName == name {
				// Remove from active servers
				s.activeServers.ActiveServers = append(
					s.activeServers.ActiveServers[:i],
					s.activeServers.ActiveServers[i+1:]...)

				if err := s.saveActiveServers(); err != nil {
					return fmt.Errorf("failed to update active servers after deletion: %w", err)
				}
				break
			}
		}
	}

	return s.saveServerConfig()
}

// EnableServer enables an MCP server by adding it to the active servers list
func (s *MCPServerSettingsService) EnableServer(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	if !s.activeLoaded {
		if err := s.loadActiveServers(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load active servers: %w", err)
		}
	}

	// Check if server exists
	if _, exists := s.serverConfig.MCPServers[name]; !exists {
		return fmt.Errorf("server with name %s does not exist", name)
	}

	// Check if already active
	for _, active := range s.activeServers.ActiveServers {
		if active == name {
			// Already enabled
			return nil
		}
	}

	// Add to active servers
	s.activeServers.ActiveServers = append(s.activeServers.ActiveServers, name)
	return s.saveActiveServers()
}

// DisableServer disables an MCP server by removing it from the active servers list
func (s *MCPServerSettingsService) DisableServer(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.activeLoaded {
		if err := s.loadActiveServers(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load active servers: %w", err)
		}
	}

	// Find the server in the active list
	for i, active := range s.activeServers.ActiveServers {
		if active == name {
			// Remove from active servers
			s.activeServers.ActiveServers = append(
				s.activeServers.ActiveServers[:i],
				s.activeServers.ActiveServers[i+1:]...)

			return s.saveActiveServers()
		}
	}

	// Server wasn't in the active list (already disabled)
	return nil
}

// SetServerEnabled sets the enabled state in the server configuration itself
func (s *MCPServerSettingsService) SetServerEnabled(name string, enabled bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			return fmt.Errorf("failed to load server config: %w", err)
		}
	}

	// Check if server exists
	serverConfig, exists := s.serverConfig.MCPServers[name]
	if !exists {
		return fmt.Errorf("server with name %s does not exist", name)
	}

	// Update the enabled state
	serverConfig.Enabled = enabled
	s.serverConfig.MCPServers[name] = serverConfig

	return s.saveServerConfig()
}

// GetEnabledServers returns servers that are both in the active list and marked as enabled
func (s *MCPServerSettingsService) GetEnabledServers() map[string]ServerConfigWrapper {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.configLoaded {
		if err := s.loadServerConfig(); err != nil {
			slog.Error("Failed to load server config", "error", err)
		}
	}

	if !s.activeLoaded {
		if err := s.loadActiveServers(); err != nil {
			slog.Error("Failed to load active servers", "error", err)
		}
	}

	enabledServers := make(map[string]ServerConfigWrapper)

	// A server is considered enabled if it's in the active list AND its Enabled flag is true
	for _, name := range s.activeServers.ActiveServers {
		if server, exists := s.serverConfig.MCPServers[name]; exists && server.Enabled {
			enabledServers[name] = server
		}
	}

	return enabledServers
}

// GetMCPConfig returns a complete MCPServerConfig with only enabled servers
func (s *MCPServerSettingsService) GetMCPConfig() *MCPServerConfig {
	enabledServers := s.GetEnabledServers()

	config := NewMCPServerConfig()
	for name, server := range enabledServers {
		config.MCPServers[name] = server
	}

	return config
}
