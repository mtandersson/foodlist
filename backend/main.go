package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Configure structured logging
	setupLogger()

	// Determine data directory
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}

	// Initialize event store
	eventFile := filepath.Join(dataDir, "events.jsonl")
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

	// Serve static files from frontend/dist
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../frontend/dist"
	}
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	// Start HTTP server
	bindAddr := os.Getenv("BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := bindAddr + ":" + port

	slog.Info("starting server",
		"bind_addr", bindAddr,
		"port", port,
		"address", addr,
		"websocket_endpoint", "ws://"+bindAddr+":"+port+"/ws",
		"static_dir", staticDir,
	)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		// defer will close store
	}
}

// setupLogger configures the global logger based on LOG_FORMAT environment variable
// Supported formats: "logfmt" (default) or "json"
func setupLogger() {
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "logfmt"
	}

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
