package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	BindAddr  string `env:"BIND_ADDR" envDefault:"localhost"`
	Port      string `env:"PORT" envDefault:"8080"`
	StaticDir string `env:"STATIC_DIR" envDefault:"../frontend/dist"`

	// Data configuration
	DataDir string `env:"DATA_DIR" envDefault:"."`

	// Logging configuration
	LogFormat string `env:"LOG_FORMAT" envDefault:"logfmt"`
}

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Parse configuration from environment variables
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		slog.Error("failed to parse configuration", "error", err)
		os.Exit(1)
	}

	// Configure structured logging
	setupLogger(cfg.LogFormat)

	// Initialize event store
	eventFile := filepath.Join(cfg.DataDir, "events.jsonl")
	absEventFile, _ := filepath.Abs(eventFile)
	slog.Info("initializing event store", "file", absEventFile)

	store, err := NewEventStore(eventFile)
	if err != nil {
		slog.Error("failed to initialize event store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Create server and load existing events
	server := NewServer(store)
	if err := server.LoadEvents(); err != nil {
		slog.Error("failed to load events", "error", err)
		return // defer will close store
	}

	// Start server event loop
	go server.Run()

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.HandleWebSocket)

	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir(cfg.StaticDir)))

	// Start HTTP server
	addr := cfg.BindAddr + ":" + cfg.Port

	slog.Info("starting server",
		"bind_addr", cfg.BindAddr,
		"port", cfg.Port,
		"address", addr,
		"websocket_endpoint", "ws://"+cfg.BindAddr+":"+cfg.Port+"/ws",
		"static_dir", cfg.StaticDir,
		"data_dir", cfg.DataDir,
	)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		// defer will close store
	}
}

// setupLogger configures the global logger based on the provided format
// Supported formats: "logfmt" (default) or "json"
func setupLogger(logFormat string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
		slog.SetDefault(slog.New(handler))
		slog.Info("logger configured", "format", "json")
	case "logfmt":
		handler = slog.NewTextHandler(os.Stdout, opts)
		slog.SetDefault(slog.New(handler))
		slog.Info("logger configured", "format", "logfmt")
	default:
		// Default to logfmt for unknown formats
		handler = slog.NewTextHandler(os.Stdout, opts)
		slog.SetDefault(slog.New(handler))
		slog.Warn("unknown log format, defaulting to logfmt", "requested_format", logFormat, "format", "logfmt")
	}
}
