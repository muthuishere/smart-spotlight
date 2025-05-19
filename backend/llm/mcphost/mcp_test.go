package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"smart-spotlight-ai/backend/packages/llm/history"
	"testing"
	"time"
)

/* pretty-print helper */
func pp(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
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
