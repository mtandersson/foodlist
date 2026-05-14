package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// version is set at build time via ldflags: -X main.version=<version>
// Default value is "dev" if not set during build
var version = "dev"

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

	// Security configuration
	SharedSecret    string   `env:"SHARED_SECRET" envDefault:""`
	CIDRWhitelist   []string `env:"CIDR_WHITELIST" envSeparator:","`
	ProxyTrustCount int      `env:"PROXY_TRUST_COUNT" envDefault:"0"`

	// Optional bearer-token HTTP API (see api.go)
	APIToken string `env:"FOODLIST_API_TOKEN" envDefault:""`

	// Embedding cache configuration. If GeminiAPIKey is empty the cache
	// build is skipped entirely.
	GeminiAPIKey       string `env:"GEMINI_API_KEY" envDefault:""`
	EmbeddingModel     string `env:"EMBEDDING_MODEL" envDefault:"gemini-embedding-001"`
	EmbeddingCacheFile string `env:"EMBEDDING_CACHE_FILE" envDefault:""`
	EmbeddingBatchSize int    `env:"EMBEDDING_BATCH_SIZE" envDefault:"100"`
	EmbeddingRPM       int    `env:"EMBEDDING_RPM" envDefault:"60"`
}

func main() {
	if len(os.Args) > 1 {
		runCLI()
		return
	}
	runHTTPServer()
}

func runHTTPServer() {
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

	// Build the embedding cache before serving traffic. The next PR will use
	// it to auto-categorize new items, so the system must not become active
	// until every uncached item has been embedded and persisted.
	if cfg.GeminiAPIKey != "" {
		cachePath := cfg.EmbeddingCacheFile
		if cachePath == "" {
			cachePath = filepath.Join(cfg.DataDir, "embeddings.jsonl")
		}
		absCachePath, _ := filepath.Abs(cachePath)
		slog.Info("initializing embedding cache",
			"file", absCachePath,
			"model", cfg.EmbeddingModel,
			"batch_size", cfg.EmbeddingBatchSize,
			"rpm", cfg.EmbeddingRPM,
		)
		cache, err := NewEmbeddingCache(cachePath)
		if err != nil {
			slog.Error("failed to initialize embedding cache", "error", err)
			return
		}
		defer cache.Close()
		builderCfg := embeddingBuilderConfig{
			Model:     cfg.EmbeddingModel,
			APIKey:    cfg.GeminiAPIKey,
			BatchSize: cfg.EmbeddingBatchSize,
			RPM:       cfg.EmbeddingRPM,
		}
		if err := BuildEmbeddingCache(context.Background(), builderCfg, server, cache); err != nil {
			slog.Error("failed to build embedding cache", "error", err)
			return
		}
		server.SetEmbeddingCache(cache)
	} else {
		slog.Info("embedding cache disabled (no GEMINI_API_KEY)")
	}

	// Start server event loop
	go server.Run()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Determine path prefix based on shared secret
	pathPrefix := "/"
	if cfg.SharedSecret != "" {
		pathPrefix = "/" + cfg.SharedSecret + "/"
	}

	// WebSocket endpoint
	wsPath := pathPrefix + "ws"
	mux.HandleFunc(wsPath, server.HandleWebSocket)

	// Also register WebSocket at /ws for whitelisted IPs (middleware will handle security)
	// This is needed because WebSocket connections can't follow HTTP redirects
	if cfg.SharedSecret != "" && len(cfg.CIDRWhitelist) > 0 {
		mux.HandleFunc("/ws", server.HandleWebSocket)
	}

	// Serve PWA files at root level (always accessible, middleware handles security)
	// These files need to be accessible from root for service worker registration
	// and mobile app installation to work properly
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "sw.js"))
	})
	mux.HandleFunc("/sw.js.map", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "sw.js.map"))
	})
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "manifest.json"))
	})
	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "manifest.webmanifest"))
	})
	mux.HandleFunc("/registerSW.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "registerSW.js"))
	})
	// Handle workbox files dynamically (they have hashed names)
	mux.HandleFunc("/workbox-", func(w http.ResponseWriter, r *http.Request) {
		// Extract filename from path
		filename := filepath.Base(r.URL.Path)
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, filename))
	})

	// Serve static files under secret path
	staticPath := pathPrefix
	fileServer := http.FileServer(http.Dir(cfg.StaticDir))
	if pathPrefix != "/" {
		// Strip the secret prefix before serving files
		// pathPrefix is like "/dev/", we need to strip "/dev" (without trailing slash)
		prefixToStrip := pathPrefix[:len(pathPrefix)-1]
		fileServer = http.StripPrefix(prefixToStrip, fileServer)
		// Also handle requests without trailing slash by redirecting
		mux.HandleFunc(pathPrefix[:len(pathPrefix)-1], func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, pathPrefix, http.StatusMovedPermanently)
		})
	}
	mux.Handle(staticPath, fileServer)

	mcpHandler := foodlistMCPHandler(server)
	mux.Handle(mcpHTTPPath, mcpHandler)
	mux.Handle(mcpHTTPPath+"/", mcpHandler)
	slog.Info("MCP streamable HTTP", "path", mcpHTTPPath)

	mux.Handle("/api/v1/state", apiBearerAuth(cfg.APIToken, http.HandlerFunc(server.handleAPIState)))
	mux.Handle("/api/v1/command", apiBearerAuth(cfg.APIToken, http.HandlerFunc(server.handleAPICommand)))
	if cfg.APIToken != "" {
		slog.Info("HTTP API enabled", "get_state", "/api/v1/state", "post_command", "/api/v1/command")
	}

	// Build middleware chain
	var handler http.Handler = mux

	// Wrap with IP whitelist middleware if configured
	if cfg.SharedSecret != "" && len(cfg.CIDRWhitelist) > 0 {
		slog.Info("IP whitelist enabled",
			"cidr_whitelist", cfg.CIDRWhitelist,
			"secret_path", pathPrefix,
		)
		handler = IPWhitelistMiddleware(cfg.CIDRWhitelist, cfg.SharedSecret, cfg.ProxyTrustCount, handler)
	} else if cfg.SharedSecret != "" {
		slog.Warn("shared secret configured but no CIDR whitelist - security features partially disabled")
	}

	// Wrap with HTTP logging middleware (outermost - logs all requests)
	handler = HTTPLoggingMiddleware(handler)

	// Start HTTP server
	addr := cfg.BindAddr + ":" + cfg.Port

	slog.Info("starting server",
		"bind_addr", cfg.BindAddr,
		"port", cfg.Port,
		"address", addr,
		"websocket_endpoint", "ws://"+cfg.BindAddr+":"+cfg.Port+wsPath,
		"static_dir", cfg.StaticDir,
		"data_dir", cfg.DataDir,
		"path_prefix", pathPrefix,
	)

	if err := http.ListenAndServe(addr, handler); err != nil {
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
