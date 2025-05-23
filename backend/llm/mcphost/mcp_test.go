package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"smart-spotlight-ai/backend/packages/llm/history"
	"testing"
	"time"
)

/* pretty-print helper */
func pp(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func TestMCPConfigUnmarshalJSON(t *testing.T) {
	// Sample JSON that includes both server types
	jsonData := `{
		"mcpServers": {
			"file_server": {
				"command": "/usr/bin/node",
				"args": ["index.js"],
				"env": {
					"DEBUG": "true",
					"PORT": "8080"
				}
			},
			"api_server": {
				"url": "https://api.example.com/mcp",
				"headers": [
					"Authorization: Bearer token123",
					"Content-Type: application/json"
				]
			}
		}
	}`

	var config MCPConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal MCPConfig: %v", err)
	}

	// Verify STDIO server configuration
	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.MCPServers))
	}

	// Check file_server (STDIO type)
	if server, ok := config.MCPServers["file_server"]; ok {
		if server.Config.GetType() != transportStdio {
			t.Errorf("Expected file_server to be %s type, got %s", transportStdio, server.Config.GetType())
		}

		stdioConfig, ok := server.Config.(STDIOServerConfig)
		if !ok {
			t.Fatalf("Failed to cast to STDIOServerConfig")
		}

		// Verify STDIO config fields
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
	} else {
		t.Errorf("file_server not found in config")
	}

	// Check api_server (SSE type)
	if server, ok := config.MCPServers["api_server"]; ok {
		if server.Config.GetType() != transportSSE {
			t.Errorf("Expected api_server to be %s type, got %s", transportSSE, server.Config.GetType())
		}

		sseConfig, ok := server.Config.(SSEServerConfig)
		if !ok {
			t.Fatalf("Failed to cast to SSEServerConfig")
		}

		// Verify SSE config fields
		if sseConfig.Url != "https://api.example.com/mcp" {
			t.Errorf("Expected URL to be 'https://api.example.com/mcp', got '%s'", sseConfig.Url)
		}

		expectedHeaders := []string{"Authorization: Bearer token123", "Content-Type: application/json"}
		if !reflect.DeepEqual(sseConfig.Headers, expectedHeaders) {
			t.Errorf("Headers mismatch, expected %v, got %v", expectedHeaders, sseConfig.Headers)
		}
	} else {
		t.Errorf("api_server not found in config")
	}

	// Test round trip marshalling/unmarshalling
	marshalled, err := json.Marshal(&config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var roundTripConfig MCPConfig
	err = json.Unmarshal(marshalled, &roundTripConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal round-trip config: %v", err)
	}

	// Print the marshalled JSON for debugging
	t.Logf("Round-trip JSON: %s", string(marshalled))
}

func TestMCPService(t *testing.T) {
	// ── 1. set up slog for this test ──────────────────────────────────────────
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug, // verbose for CI
		}),
	)
	slog.SetDefault(logger)

	// ── 2. read env vars ─────────────────────────────────────────────────────
	providerName := os.Getenv("SPOT_AI_PROVIDER")
	baseURL := os.Getenv("SPOT_AI_API_ENDPOINT")
	apiKey := os.Getenv("SPOT_AI_API_KEY")
	modelName := os.Getenv("SPOT_AI_MODEL")
	systemPrompt := os.Getenv("SPOT_AI_SYSTEM_PROMPT")
	cfgFile := os.Getenv("SPOT_AI_CONFIG_FILE")

	if apiKey == "" {
		logger.Warn("API key missing; skipping integration test")
		t.Skip("SPOT_AI_API_KEY not set")
	}

	// ── 3. build service settings ────────────────────────────────────────────
	set := &MCPSettings{
		ConfigFile:    cfgFile,
		SystemPrompt:  systemPrompt,
		MessageWindow: 10,
		Provider: LLMProvider{
			ProviderName: providerName,
			BaseURL:      baseURL,
			APIKey:       apiKey,
			ModelName:    modelName,
			Metadata:     map[string]string{},
		},
		DebugMode: true,
	}
	logger.Info("constructed MCPSettings", "config", pp(set))

	// ── 4. init service ──────────────────────────────────────────────────────
	svc, err := NewMCPService(set)
	if err != nil {
		t.Fatalf("NewMCPService: %v", err)
	}
	logger.Debug("NewMCPService done")

	if err := svc.InitializeClients(); err != nil {
		t.Fatalf("InitializeClients: %v", err)
	}
	logger.Debug("InitializeClients done", "toolCount", len(svc.tools))

	// ── 5. start prompt loop (async) ─────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartPromptLoop(ctx)
	logger.Debug("prompt loop started")

	/* utility waits for EventFinalResult, logging all events in-between */
	waitForAnswer := func(q string) (history.HistoryMessage, error) {
		logger.Info("sending prompt", "query", q)
		if err := svc.Search(q); err != nil {
			return history.HistoryMessage{}, err
		}

		timeout := time.After(45 * time.Second)
		for {
			select {
			case ev := <-svc.EventChan:
				logger.Debug("got PromptEvent", "type", ev.Type)
				switch ev.Type {
				case EventError:
					return history.HistoryMessage{}, fmt.Errorf("backend error: %v", ev.Data)
				case EventFinalResult:
					return ev.Data.(history.HistoryMessage), nil
				}
			case <-timeout:
				return history.HistoryMessage{}, fmt.Errorf("timeout waiting for final_result")
			}
		}
	}

	// ── 6. run sub-test ──────────────────────────────────────────────────────
	t.Run("file-query", func(t *testing.T) {
		msg, err := waitForAnswer("How many tables are there in the database?")
		if err != nil {
			t.Fatalf("waitForAnswer: %v", err)
		}
		logger.Info("assistant final reply", "content", pp(msg))

		if len(msg.Content) == 0 {
			t.Fatalf("empty content in final_result")
		}
	})
}
