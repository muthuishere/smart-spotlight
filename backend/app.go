package backend

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"

	"smart-spotlight-wails/backend/history"
	"smart-spotlight-wails/backend/keybind"
	"smart-spotlight-wails/backend/llm"
	"smart-spotlight-wails/backend/llm/mcphost"

	llmhistory "smart-spotlight-wails/backend/packages/llm/history"
	"smart-spotlight-wails/backend/settings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	version         string
	db              *sql.DB
	startupComplete bool
	historyService  *history.Service
	llmService      *llm.Service
	mcpService      *mcphost.MCPService
}

// NewApp creates a new App application struct
func NewApp() *App {

	// slog.Info("Starting application")
	// for _, env := range os.Environ() {
	// 	//if strings.HasPrefix(strings.ToUpper(env), "APP") {
	// 	slog.Info("Environment variable", "value", env)
	// 	//		}
	// }

	return &App{
		startupComplete: false,
	}
}

// SetVersion sets the app version
func (a *App) SetVersion(version string) {
	a.version = version
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	settings.InitSettings()

	configDir, err := settings.GetConfigDir()
	if err != nil {
		log.Printf("Error getting config directory: %v", err)
		return
	}

	// Initialize database
	dbPath := filepath.Join(configDir, "history.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database: %v", err)
	}
	a.db = db

	// Initialize services
	a.historyService = history.NewService(db)
	if err := a.historyService.Initialize(); err != nil {
		log.Printf("Error initializing search history: %v", err)
	}

	a.llmService = llm.NewService(settings.GetCurrentSettings())

	// Initialize MCP service
	if err := a.initializeMCPService(); err != nil {
		log.Printf("Error initializing MCP service: %v", err)
	}

	// Setup global shortcut
	a.setupGlobalShortcut()
	a.startupComplete = true
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// DomReady is called after the front-end dom has been loaded
func (a *App) DomReady(ctx context.Context) {
	// Center the window on startup
	runtime.WindowCenter(ctx)
	// Hide the window initially
	runtime.WindowHide(ctx)
}

// setupGlobalShortcut registers the global shortcut
func (a *App) setupGlobalShortcut() {
	hk := keybind.GetHotkey()
	if err := hk.Register(); err == nil {
		go func() {
			for range hk.Keydown() {
				// Toggle window visibility
				isVisible := runtime.WindowIsNormal(a.ctx)
				if isVisible {
					runtime.WindowHide(a.ctx)
				} else {
					keybind.ShowWindow(a.ctx)
					runtime.WindowCenter(a.ctx) // Center the window when showing
				}
			}
		}()
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's a new Incremental show time!", name)
}

// UpdateSettings updates the application settings
func (a *App) UpdateSettings(newSettings settings.Settings) error {
	err := settings.SaveSettings(&newSettings)
	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	settings.AppSettings = &newSettings         // Update in-memory settings
	a.llmService = llm.NewService(&newSettings) // Update LLM service with new settings
	return nil
}

// GetSettings retrieves the current application settings
func (a *App) GetSettings() *settings.Settings {
	return settings.AppSettings
}

// SearchWithLLM performs a search using the LLM service
func (a *App) SearchWithLLM(query string) (*llm.ChatResponse, error) {
	if err := a.historyService.AddToHistory(query); err != nil {
		log.Printf("Error adding to history: %v", err)
	}
	return a.llmService.Search(query)
}

// SearchWithMCP performs a search using the MCP service
func (a *App) SearchWithMCP(query string) error {
	if a.mcpService == nil {
		return fmt.Errorf("MCP service is not initialized")
	}

	// Add query to history
	if err := a.historyService.AddToHistory(query); err != nil {
		log.Printf("Error adding to history: %v", err)
	}

	// Perform search using MCP service
	return a.mcpService.Search(query)
}

// GetSearchHistory returns the search history
func (a *App) GetSearchHistory(prefix string) []history.SearchHistory {
	return a.historyService.GetSearchHistory(prefix)
}

// TestAPIConnection tests if the API settings are valid
func (a *App) TestAPIConnection() error {
	return a.llmService.TestAPIConnection()
}

func (a *App) IsStartupComplete() bool {
	return a.startupComplete
}

// initializeMCPService initializes the MCP service with configurations from environment variables
func (a *App) initializeMCPService() error {
	// Load configuration from environment variables
	providerName := settings.GetEnvWithDefault("SPOT_AI_PROVIDER", "openai")
	baseURL := settings.GetEnvWithDefault("SPOT_AI_API_ENDPOINT", "")
	apiKey := settings.GetEnvWithDefault("SPOT_AI_API_KEY", "")
	modelName := settings.GetEnvWithDefault("SPOT_AI_MODEL", "gpt-4o")
	systemPrompt := settings.GetEnvWithDefault("SPOT_AI_SYSTEM_PROMPT", "")
	configFile := settings.GetEnvWithDefault("SPOT_AI_CONFIG_FILE", "")

	// Default to 10 if not specified
	messageWindow := 10

	// Skip initialization if API key is not provided
	if apiKey == "" {
		log.Printf("Warning: SPOT_AI_API_KEY not set, MCP service will not be initialized")
		return nil
	}

	// Create provider configuration
	provider := mcphost.LLMProvider{
		ProviderName: providerName,
		BaseURL:      baseURL,
		APIKey:       apiKey,
		ModelName:    modelName,
		Metadata:     make(map[string]string),
	}

	debugMode := settings.GetEnvWithDefault("SPOT_AI_DEBUG", "true")
	debugModeBool := debugMode == "true"
	// Create MCP settings
	mcpSettings := &mcphost.MCPSettings{
		ConfigFile:    configFile,
		SystemPrompt:  systemPrompt,
		MessageWindow: messageWindow,
		Provider:      provider,
		DebugMode:     debugModeBool, // Use app's debug setting
	}

	// Create MCP service instance
	var err error
	a.mcpService, err = mcphost.NewMCPService(a.ctx, mcpSettings)
	if err != nil {
		return fmt.Errorf("failed to create MCP service: %w", err)
	}

	if err := a.mcpService.InitializeClients(); err != nil {
		return fmt.Errorf("failed to init MCP clients: %w", err)
	}

	// 2. start the background loop so InputChan has a receiver
	a.mcpService.StartPromptLoop(a.ctx)

	// backend/app.go  (inside initializeMCPService – replace the forwarder)

	go func() {
		for ev := range a.mcpService.EventChan {

			// make a browser-friendly copy
			out := map[string]interface{}{
				"Type": ev.Type,
			}

			switch ev.Type {

			case mcphost.EventFinalResult:
				// extract plain markdown
				if msg, ok := ev.Data.(llmhistory.HistoryMessage); ok {
					txt := ""
					for _, b := range msg.Content {
						if b.Type == "text" {
							txt = b.Text
							break
						}
					}
					out["Data"] = txt
				}

			case mcphost.EventToolUse,
				mcphost.EventToolResult,
				mcphost.EventAuthorization,
				mcphost.EventConfirmationRequired:
				out["Data"] = ev.Data // these are already maps / strings

			case mcphost.EventError:
				out["Data"] = fmt.Sprintf("%v", ev.Data)
			}

			slog.Info("Emit to frontend", "payload", out)
			a.mcpService.EmitPublic("PromptEvent", ev.Type, out)
			// runtime.EventsEmit(a.ctx, "PromptEvent", out)
		}
	}()

	return nil
}

func (a *App) ConfirmTool(token string, ok bool) error {
	if a.mcpService != nil {
		a.mcpService.Confirm(token, ok)
	}
	return nil
}
