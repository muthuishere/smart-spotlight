// filepath: /Users/muthuishere/muthu/gitworkspace/mcp/smar-spotlight-mcp-host/backend/settings/mcp_settings_service_test.go
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMCPServerSettingsServiceE2E(t *testing.T) {
	// Create a temp directory for this test to avoid polluting the real config
	tempDir, err := os.MkdirTemp("", "mcp_settings_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up a test MCPServerSettingsService that uses our temp directory
	service, err := createTestMCPServerSettingsService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Run a full lifecycle test of CRUD operations
	t.Run("Complete CRUD Lifecycle", func(t *testing.T) {
		// 1. Initially, there should be no servers
		servers := service.GetAllServers()
		if len(servers) != 0 {
			t.Fatalf("Expected 0 servers initially, got %d", len(servers))
		}

		// 2. Add a STDIO server
		err := service.AddSTDIOServer(
			"file_server",
			"/usr/bin/node",
			[]string{"index.js"},
			map[string]string{"DEBUG": "true", "PORT": "8080"},
		)
		if err != nil {
			t.Fatalf("Failed to add STDIO server: %v", err)
		}

		// 3. Add an SSE server
		err = service.AddSSEServer(
			"api_server",
			"https://api.example.com/mcp",
			[]string{"Authorization: Bearer token123", "Content-Type: application/json"},
		)
		if err != nil {
			t.Fatalf("Failed to add SSE server: %v", err)
		}

		// 4. Verify both servers were added
		servers = service.GetAllServers()
		if len(servers) != 2 {
			t.Fatalf("Expected 2 servers after adding, got %d", len(servers))
		}

		// 5. Verify STDIO server configuration
		stdioServer, exists := servers["file_server"]
		if !exists {
			t.Fatalf("file_server not found in servers")
		}
		if stdioServer.Config.GetType() != transportStdio {
			t.Errorf("Expected file_server to be %s type, got %s", transportStdio, stdioServer.Config.GetType())
		}

		// Cast to check specific fields
		stdioConfig, ok := stdioServer.Config.(STDIOServerConfig)
		if !ok {
			t.Fatalf("Failed to cast to STDIOServerConfig")
		}
		if stdioConfig.Command != "/usr/bin/node" {
			t.Errorf("Expected command to be '/usr/bin/node', got '%s'", stdioConfig.Command)
		}
		if len(stdioConfig.Args) != 1 || stdioConfig.Args[0] != "index.js" {
			t.Errorf("Args mismatch, got %v", stdioConfig.Args)
		}
		expectedEnv := map[string]string{"DEBUG": "true", "PORT": "8080"}
		if !reflect.DeepEqual(stdioConfig.Env, expectedEnv) {
			t.Errorf("Env mismatch, expected %v, got %v", expectedEnv, stdioConfig.Env)
		}

		// 6. Verify SSE server configuration
		sseServer, exists := servers["api_server"]
		if !exists {
			t.Fatalf("api_server not found in servers")
		}
		if sseServer.Config.GetType() != transportSSE {
			t.Errorf("Expected api_server to be %s type, got %s", transportSSE, sseServer.Config.GetType())
		}

		// Cast to check specific fields
		sseConfig, ok := sseServer.Config.(SSEServerConfig)
		if !ok {
			t.Fatalf("Failed to cast to SSEServerConfig")
		}
		if sseConfig.Url != "https://api.example.com/mcp" {
			t.Errorf("Expected URL to be 'https://api.example.com/mcp', got '%s'", sseConfig.Url)
		}
		expectedHeaders := []string{"Authorization: Bearer token123", "Content-Type: application/json"}
		if !reflect.DeepEqual(sseConfig.Headers, expectedHeaders) {
			t.Errorf("Headers mismatch, expected %v, got %v", expectedHeaders, sseConfig.Headers)
		}

		// 7. Enable the STDIO server
		err = service.EnableServer("file_server")
		if err != nil {
			t.Fatalf("Failed to enable file_server: %v", err)
		}

		// 8. Verify only the STDIO server is active
		activeServers := service.GetActiveServers()
		if len(activeServers) != 1 {
			t.Fatalf("Expected 1 active server, got %d", len(activeServers))
		}
		if _, exists := activeServers["file_server"]; !exists {
			t.Errorf("file_server should be active but isn't")
		}

		// 9. Enable the SSE server
		err = service.EnableServer("api_server")
		if err != nil {
			t.Fatalf("Failed to enable api_server: %v", err)
		}

		// 10. Verify both servers are active
		activeServers = service.GetActiveServers()
		if len(activeServers) != 2 {
			t.Fatalf("Expected 2 active servers, got %d", len(activeServers))
		}

		// 11. Disable the STDIO server
		err = service.DisableServer("file_server")
		if err != nil {
			t.Fatalf("Failed to disable file_server: %v", err)
		}

		// 12. Verify only the SSE server is active
		activeServers = service.GetActiveServers()
		if len(activeServers) != 1 {
			t.Fatalf("Expected 1 active server, got %d", len(activeServers))
		}
		if _, exists := activeServers["api_server"]; !exists {
			t.Errorf("api_server should be active but isn't")
		}

		// 13. Update the SSE server
		sseServer.Config = SSEServerConfig{
			Url:     "https://updated-api.example.com/mcp",
			Headers: []string{"Authorization: Bearer updated-token", "Content-Type: application/json"},
		}
		err = service.UpdateServer("api_server", sseServer)
		if err != nil {
			t.Fatalf("Failed to update api_server: %v", err)
		}

		// 14. Verify the update
		updatedServer, exists := service.GetServer("api_server")
		if !exists {
			t.Fatalf("api_server not found after update")
		}
		updatedSseConfig, ok := updatedServer.Config.(SSEServerConfig)
		if !ok {
			t.Fatalf("Failed to cast to SSEServerConfig after update")
		}
		if updatedSseConfig.Url != "https://updated-api.example.com/mcp" {
			t.Errorf("URL not updated correctly, got '%s'", updatedSseConfig.Url)
		}

		// 15. Delete the STDIO server
		err = service.DeleteServer("file_server")
		if err != nil {
			t.Fatalf("Failed to delete file_server: %v", err)
		}

		// 16. Verify only the SSE server remains
		remainingServers := service.GetAllServers()
		if len(remainingServers) != 1 {
			t.Fatalf("Expected 1 server after deletion, got %d", len(remainingServers))
		}
		if _, exists := remainingServers["api_server"]; !exists {
			t.Errorf("api_server should exist but doesn't")
		}
		if _, exists := remainingServers["file_server"]; exists {
			t.Errorf("file_server should have been deleted but still exists")
		}

		// 17. Test SetServerEnabled (which differs from Enable/Disable)
		err = service.SetServerEnabled("api_server", false)
		if err != nil {
			t.Fatalf("Failed to set api_server enabled state: %v", err)
		}

		// 18. Verify the server is still in the active list but has enabled=false
		updatedServer, exists = service.GetServer("api_server")
		if !exists {
			t.Fatalf("api_server not found after setting enabled state")
		}
		if updatedServer.Enabled {
			t.Errorf("api_server should have enabled=false but has enabled=true")
		}

		// It should still be in the active list
		activeServers = service.GetActiveServers()
		if _, exists := activeServers["api_server"]; !exists {
			t.Errorf("api_server should still be in the active list")
		}

		// But it shouldn't be in the GetEnabledServers result
		enabledServers := service.GetEnabledServers()
		if len(enabledServers) != 0 {
			t.Errorf("Expected 0 enabled servers, got %d", len(enabledServers))
		}

		// 19. Re-enable and verify it's properly enabled
		err = service.SetServerEnabled("api_server", true)
		if err != nil {
			t.Fatalf("Failed to re-enable api_server: %v", err)
		}

		enabledServers = service.GetEnabledServers()
		if len(enabledServers) != 1 {
			t.Errorf("Expected 1 enabled server, got %d", len(enabledServers))
		}

		// 20. Verify file persistence by reloading
		newService, err := createTestMCPServerSettingsService(tempDir)
		if err != nil {
			t.Fatalf("Failed to create new test service: %v", err)
		}

		persistedServers := newService.GetAllServers()
		if len(persistedServers) != 1 {
			t.Fatalf("Expected 1 persisted server, got %d", len(persistedServers))
		}
		if _, exists := persistedServers["api_server"]; !exists {
			t.Errorf("api_server should exist in persisted data but doesn't")
		}

		persistedActiveServers := newService.GetActiveServerNames()
		if len(persistedActiveServers) != 1 {
			t.Fatalf("Expected 1 persisted active server, got %d", len(persistedActiveServers))
		}
		if persistedActiveServers[0] != "api_server" {
			t.Errorf("Expected api_server in active servers, got %s", persistedActiveServers[0])
		}

		// 21. Test GetMCPConfig which should give us a complete config
		mcpConfig := newService.GetMCPConfig()
		if len(mcpConfig.MCPServers) != 1 {
			t.Fatalf("Expected 1 server in MCPConfig, got %d", len(mcpConfig.MCPServers))
		}
		if _, exists := mcpConfig.MCPServers["api_server"]; !exists {
			t.Errorf("api_server should be in MCPConfig but isn't")
		}
	})

	// Test file paths
	t.Run("File Paths", func(t *testing.T) {
		expectedServersPath := filepath.Join(tempDir, mcpServersFileName)
		expectedActivePath := filepath.Join(tempDir, activeMCPServersFileName)

		if service.GetServersFilePath() != expectedServersPath {
			t.Errorf("Expected servers file path %s, got %s", expectedServersPath, service.GetServersFilePath())
		}

		if service.GetActiveServersFilePath() != expectedActivePath {
			t.Errorf("Expected active servers file path %s, got %s", expectedActivePath, service.GetActiveServersFilePath())
		}

		// Verify files exist
		if _, err := os.Stat(expectedServersPath); os.IsNotExist(err) {
			t.Errorf("Servers file doesn't exist: %s", expectedServersPath)
		}
		if _, err := os.Stat(expectedActivePath); os.IsNotExist(err) {
			t.Errorf("Active servers file doesn't exist: %s", expectedActivePath)
		}

		// Check that the files contain valid JSON
		serversBytes, err := os.ReadFile(expectedServersPath)
		if err != nil {
			t.Fatalf("Failed to read servers file: %v", err)
		}
		var mcpConfig MCPServerConfig
		if err := json.Unmarshal(serversBytes, &mcpConfig); err != nil {
			t.Fatalf("Servers file doesn't contain valid JSON: %v", err)
		}

		activeBytes, err := os.ReadFile(expectedActivePath)
		if err != nil {
			t.Fatalf("Failed to read active servers file: %v", err)
		}
		var activeConfig ActiveMCPServers
		if err := json.Unmarshal(activeBytes, &activeConfig); err != nil {
			t.Fatalf("Active servers file doesn't contain valid JSON: %v", err)
		}
	})

	// Test concurrency safety
	t.Run("Concurrency Safety", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				// Use a unique name for each goroutine
				serverName := fmt.Sprintf("concurrent_server_%d", idx)

				// Add a server
				err := service.AddSTDIOServer(
					serverName,
					"/usr/bin/concurrent",
					[]string{"arg1", "arg2"},
					map[string]string{"IDX": serverName},
				)
				if err != nil {
					t.Logf("Failed to add server %s: %v", serverName, err)
				}

				// Enable it
				err = service.EnableServer(serverName)
				if err != nil {
					t.Logf("Failed to enable server %s: %v", serverName, err)
				}

				// Update it
				server, exists := service.GetServer(serverName)
				if exists {
					stdioConfig, ok := server.Config.(STDIOServerConfig)
					if ok {
						stdioConfig.Args = append(stdioConfig.Args, "new_arg")
						server.Config = stdioConfig
						err = service.UpdateServer(serverName, server)
						if err != nil {
							t.Logf("Failed to update server %s: %v", serverName, err)
						}
					}
				}

				// Give other goroutines a chance
				time.Sleep(time.Millisecond)

				done <- true
			}(i)
		}

		// Wait for all goroutines to finish
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check the result
		servers := service.GetAllServers()
		if len(servers) != numGoroutines+1 { // +1 for the api_server from previous test
			t.Errorf("Expected %d servers after concurrent operations, got %d", numGoroutines+1, len(servers))
		}
	})
}

// Helper function to create a test service with a custom config directory
func createTestMCPServerSettingsService(tempDir string) (*MCPServerSettingsService, error) {
	// Create a service that uses our temp directory
	return NewMCPServerSettingsService(tempDir)
}
