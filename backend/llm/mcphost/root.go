package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"smart-spotlight-ai/backend/packages/llm/history"
	"smart-spotlight-ai/backend/packages/llm/models"
	"smart-spotlight-ai/backend/packages/llm/providers/anthropic"
	"smart-spotlight-ai/backend/packages/llm/providers/google"
	"smart-spotlight-ai/backend/packages/llm/providers/ollama"
	"smart-spotlight-ai/backend/packages/llm/providers/openai"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/google/uuid"

	"github.com/mark3labs/mcp-go/mcp"
)

func createProvider(ctx context.Context, settings *MCPSettings) (models.Provider, error) {
	providerName := settings.Provider.ProviderName
	modelName := settings.Provider.ModelName
	systemPrompt := settings.SystemPrompt
	apiKey := settings.Provider.APIKey

	if apiKey == "" {
		return nil, fmt.Errorf(
			" API key not provided  variable",
		)
	}

	// Use the provider configuration
	switch providerName {
	case "openai":
		slog.Info("Creating OpenAI provider")

		return openai.NewProvider(apiKey, settings.Provider.BaseURL, modelName, systemPrompt), nil

	// You can add more provider types here as needed
	case "anthropic":

		slog.Info("Creating Anthropic provider")
		return anthropic.NewProvider(apiKey, settings.Provider.BaseURL, modelName, systemPrompt), nil

	case "ollama":
		slog.Info("Creating Ollama provider")
		return ollama.NewProvider(modelName, systemPrompt)

	case "google":
		slog.Info("Creating Google provider")

		return google.NewProvider(ctx, apiKey, modelName, systemPrompt)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// NewMCPService creates a new MCP service
func NewMCPService(ctx context.Context, settings *MCPSettings) (*MCPService, error) {
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
		ctx:            ctx,
		settings:       settings,
		provider:       provider,
		logger:         logger,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		maxRetries:     maxRetries,
		waitingConfirm: false,
		InputChan:      make(chan PromptEvent),
		EventChan:      make(chan PromptEvent),
		ConfirmChan:    make(chan confirmationReply),
	}, nil
}

// handleDirectFollowUp creates a direct follow-up prompt with tool results

func (s *MCPService) emit(ev PromptEvent) {
	if s.waitingConfirm && ev.Type != EventConfirmationRequired {
		// suppress everything except the confirmation itself
		return
	}
	s.EventChan <- ev
}

func (s *MCPService) Search(query string) error {
	s.logger.Info("Search request received",
		"query", query,
		"timestamp", time.Now().Format(time.RFC3339),
		"provider", s.settings.Provider.ProviderName,
		"model", s.settings.Provider.ModelName)

	// Log query characteristics
	s.logger.Debug("Query details",
		"query_length", len(query),
		"query_words", len(strings.Fields(query)))

	// Log channel state
	s.logger.Debug("Sending event to input channel",
		"channel_cap", cap(s.InputChan),
		"event_type", EventPrompt)

	// Create and send the event
	event := PromptEvent{Type: EventPrompt, Data: query}

	// Try to send with timeout to detect potential deadlocks
	select {
	case s.InputChan <- event:
		s.logger.Debug("Event successfully sent to input channel")
	case <-time.After(100 * time.Millisecond):
		s.logger.Warn("Channel send operation taking longer than expected, proceeding anyway")
		s.InputChan <- event
	}

	s.logger.Info("Search request queued successfully",
		"query_id", fmt.Sprintf("%x", time.Now().UnixNano()))

	return nil
}

func (s *MCPService) StartPromptLoop(ctx context.Context) {
	messages := make([]history.HistoryMessage, 0)

	go func() {
		for {
			if err := s.RunPromptWithChannels(ctx, &messages); err != nil {
				s.logger.Error("prompt execution failed", "error", err)
				s.emit(PromptEvent{Type: EventError, Data: err.Error()})
			}
		}
	}()
}

func (s *MCPService) InitializeClients() error {
	config, err := loadMCPConfig(s.settings)
	if err != nil {
		return err
	}

	clients, err := createMCPClients(config)
	if err != nil {
		return err
	}

	s.mcpClients = clients
	s.tools = []models.Tool{}

	for name, client := range clients {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			s.logger.Error("failed to fetch tools", "server", name, "error", err)
			continue
		}
		s.tools = append(s.tools, mcpToolsToAnthropicTools(name, toolsResult.Tools)...)
	}
	return nil
}

func (s *MCPService) RunPromptWithChannels(
	ctx context.Context,
	messages *[]history.HistoryMessage,
) error {
	evt := <-s.InputChan
	if evt.Type != EventPrompt {
		return nil
	}
	prompt := evt.Data.(string)
	*messages = append(*messages,
		history.HistoryMessage{Role: "user",
			Content: []history.ContentBlock{{Type: "text", Text: prompt}}})

	return s.runLLMWithToolCycle(ctx, prompt, messages)
}

// runLLMWithToolCycle makes one provider call, executes tool calls,
// inserts `tool_result` messages, and (if tools were used) recurses once
// with an empty prompt so the LLM can craft the final answer.
func (s *MCPService) runLLMWithToolCycle(
	ctx context.Context,
	prompt string,
	messages *[]history.HistoryMessage,
) error {

	/* ─ 1. Optional pruning ───────────────────────────────────────────── */
	if win := s.settings.MessageWindow; win > 0 {
		*messages = pruneMessages(*messages, win)
	}

	/* ─ 2. Adapt history to models.Message ───────────────────────────── */
	llmMsgs := make([]models.Message, len(*messages))
	for i := range *messages {
		llmMsgs[i] = &(*messages)[i]
	}

	/* ─ 3. Provider call ─────────────────────────────────────────────── */
	msg, err := s.provider.CreateMessage(ctx, prompt, llmMsgs, s.tools)
	if err != nil {
		s.emit(PromptEvent{Type: EventError, Data: err.Error()})
		return nil
	}

	/* ─ 4. Gather assistant text + tool_use blocks  ──────────────────── */
	var assistantBlocks []history.ContentBlock
	if txt := msg.GetContent(); txt != "" {
		assistantBlocks = append(assistantBlocks,
			history.ContentBlock{Type: "text", Text: txt})
	}
	for _, call := range msg.GetToolCalls() {
		rawArgs, _ := json.Marshal(call.GetArguments())
		assistantBlocks = append(assistantBlocks,
			history.ContentBlock{Type: "tool_use", ID: call.GetID(),
				Name: call.GetName(), Input: rawArgs})
	}

	/* ─ 5. ✨ Append assistant message BEFORE tool results  ───────────── */
	*messages = append(*messages, history.HistoryMessage{
		Role:    msg.GetRole(), // "assistant"
		Content: assistantBlocks,
	})

	/* ─ 6. Execute each tool call & append tool_result message ───────── */
	for _, call := range msg.GetToolCalls() {
		parts := strings.Split(call.GetName(), "__")
		if len(parts) != 2 {
			s.emit(PromptEvent{Type: EventError, Data: "invalid tool name format"})
			continue
		}
		server, tool := parts[0], parts[1]

		args := call.GetArguments() // map[string]any

		if need, token := s.confirmationRequired(server, tool, args); need {
			s.waitingConfirm = true
			argJSON, _ := json.MarshalIndent(args, "", "  ")

			s.EventChan <- PromptEvent{
				Type: EventConfirmationRequired,
				Data: map[string]any{
					"token":  token,
					"server": server,
					"tool":   tool,
					"args":   string(argJSON),
				},
			}

			// wait…
			select {
			case reply := <-s.ConfirmChan:
				s.waitingConfirm = false
				if reply.Token != token {
					// ignore mismatched confirmations (rare)
					continue
				}
				if !reply.OK {
					s.emit(PromptEvent{
						Type: EventError,
						Data: "operation aborted by user",
					})
					return nil
				}
				// user confirmed, continue
			case <-time.After(120 * time.Second):
				s.waitingConfirm = false
				s.emit(PromptEvent{
					Type: EventError,
					Data: "confirmation timeout",
				})
				return nil
			}
		}

		client := s.mcpClients[server]

		req := mcp.CallToolRequest{}               // zero-value struct
		req.Params.Name = tool                     // e.g. "list_tables"
		req.Params.Arguments = call.GetArguments() // map[string]any

		toolStart := time.Now()
		res, err := client.CallTool(ctx, req)
		toolMS := time.Since(toolStart).Milliseconds()

		slog.Debug("tool call",
			"server", server,
			"tool", tool,
			"args", call.GetArguments(),
			"duration_ms", toolMS,
			"response", res,
			"error", err,
			"timestamp", time.Now().Format(time.RFC3339))

		// Execute tool
		// res, err := client.CallTool(ctx, mcp.CallToolRequest{
		// 	Params: mcp.ToolParams{Name: tool, Arguments: call.GetArguments()},
		// })
		if err != nil {
			s.emit(PromptEvent{Type: EventError, Data: err.Error()})
			return nil
		}

		// Build tool_result block
		tr := history.ContentBlock{
			Type:      "tool_result",
			ToolUseID: call.GetID(),
			Content:   res.Content,
		}
		for _, it := range res.Content {
			if t, ok := it.(mcp.TextContent); ok {
				tr.Text = t.Text
				break
			}
		}

		// Append as its own `tool` message
		*messages = append(*messages, history.HistoryMessage{
			Role:    "tool",
			Content: []history.ContentBlock{tr},
		})
	}

	/* ─ 7. Recurse once (if any tool was used) otherwise emit final ──── */
	if len(msg.GetToolCalls()) > 0 {
		return s.runLLMWithToolCycle(ctx, "", messages)
	}

	final := (*messages)[len(*messages)-1] // last assistant message
	s.emit(PromptEvent{Type: EventFinalResult, Data: final})
	return nil
}

func (s *MCPService) Confirm(token string, ok bool) {
	s.ConfirmChan <- confirmationReply{Token: token, OK: ok}
}

func (s *MCPService) EmitPublic(evtname string, evtType, out any) {

	if s.waitingConfirm && evtType != EventConfirmationRequired {
		// suppress everything except the confirmation itself
		return
	}

	runtime.EventsEmit(s.ctx, evtname, out)
}

// -----------------------------------------------------------------------------
// Confirmation hook (stub)
// -----------------------------------------------------------------------------

func (s *MCPService) confirmationRequired(
	server string,
	tool string,
	args map[string]any, // full argument map
) (bool, string) {

	// 1. quick check on tool name
	if writeActionRe.MatchString(tool) {
		return true, uuid.NewString()
	}

	// 2. stringify *all* argument values and search again
	if writeActionRe.MatchString(fmt.Sprintf("%v", args)) {
		return true, uuid.NewString()
	}

	// nothing suspicious
	return false, ""
}

// -----------------------------------------------------------------------------
// Context pruning (unchanged)
// -----------------------------------------------------------------------------

func pruneMessages(msgs []history.HistoryMessage, window int) []history.HistoryMessage {
	if len(msgs) <= window {
		return msgs
	}

	pruned := msgs[len(msgs)-window:]
	useIDs := map[string]bool{}
	resIDs := map[string]bool{}

	for _, m := range pruned {
		for _, b := range m.Content {
			if b.Type == "tool_use" {
				useIDs[b.ID] = true
			} else if b.Type == "tool_result" {
				resIDs[b.ToolUseID] = true
			}
		}
	}

	filtered := make([]history.HistoryMessage, 0, len(pruned))
	for _, m := range pruned {
		var blocks []history.ContentBlock
		for _, b := range m.Content {
			keep := true
			if b.Type == "tool_use" {
				keep = resIDs[b.ID]
			} else if b.Type == "tool_result" {
				keep = useIDs[b.ToolUseID]
			}
			if keep {
				blocks = append(blocks, b)
			}
		}
		if len(blocks) > 0 || m.Role != "assistant" {
			m.Content = blocks
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func truncateString(s string, maxLength int) string {
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}
