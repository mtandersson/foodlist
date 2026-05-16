package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// validateCachePath resolves cacheFile to an absolute path and confirms it
// is contained inside dataDir. Returns the resolved absolute path on
// success. The check uses a trailing separator so /data/foo never matches
// /data/foobar. This is the path-traversal guard from the workspace rules.
func validateCachePath(cacheFile, dataDir string) (string, error) {
	absCache, err := filepath.Abs(cacheFile)
	if err != nil {
		return "", fmt.Errorf("resolve cache path: %w", err)
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	cacheWithSep := absCache + string(os.PathSeparator)
	dataWithSep := absData + string(os.PathSeparator)
	if !strings.HasPrefix(cacheWithSep, dataWithSep) {
		return "", fmt.Errorf("cache path %q escapes data dir %q", absCache, absData)
	}
	return absCache, nil
}

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

	// Auto-categorize via embeddings. All tunables have defaults that
	// match the documented algorithm; see backend/categorizer.go.
	EmbeddingCategorizeEnabled             bool    `env:"EMBEDDING_CATEGORIZE_ENABLED" envDefault:"true"`
	EmbeddingCategorizeSimilarityFloor     float32 `env:"EMBEDDING_CATEGORIZE_SIMILARITY_FLOOR" envDefault:"0.55"`
	EmbeddingCategorizeRecencyWindowDays   int     `env:"EMBEDDING_CATEGORIZE_RECENCY_WINDOW_DAYS" envDefault:"30"`
	EmbeddingCategorizeRecentWeight        float32 `env:"EMBEDDING_CATEGORIZE_RECENT_WEIGHT" envDefault:"0.70"`
	EmbeddingCategorizePopularityWeight    float32 `env:"EMBEDDING_CATEGORIZE_POPULARITY_WEIGHT" envDefault:"0.30"`
	EmbeddingCategorizeMaxSimGate          float32 `env:"EMBEDDING_CATEGORIZE_MAX_SIM_GATE" envDefault:"0.20"`
	EmbeddingCategorizeAcceptanceThreshold float32 `env:"EMBEDDING_CATEGORIZE_ACCEPTANCE_THRESHOLD" envDefault:"0.30"`

	// Suggestion engine. Requires embeddings to be active; if
	// GEMINI_API_KEY is unset the feature is forcibly disabled regardless
	// of this flag.
	SuggestionsEnabled         bool    `env:"SUGGESTIONS_ENABLED" envDefault:"true"`
	SuggestionsMinPurchases    int     `env:"SUGGESTIONS_MIN_PURCHASES" envDefault:"3"`
	SuggestionsMaxIntervalDays int     `env:"SUGGESTIONS_MAX_INTERVAL_DAYS" envDefault:"90"`
	SuggestionsDueFraction     float32 `env:"SUGGESTIONS_DUE_FRACTION" envDefault:"0.667"`
	SuggestionsDedupSimilarity float32 `env:"SUGGESTIONS_DEDUP_SIMILARITY" envDefault:"0.85"`
	SuggestionsRecentLimit     int     `env:"SUGGESTIONS_RECENT_PURCHASES_LIMIT" envDefault:"6"`
	SuggestionsRecomputeHours  int     `env:"SUGGESTIONS_RECOMPUTE_INTERVAL_HOURS" envDefault:"6"`
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

	// Build the embedding cache before serving traffic, and share the same
	// embedding client with the runtime auto-categorize hook so we only
	// maintain one RPM bucket.
	if cfg.GeminiAPIKey != "" {
		cachePath := cfg.EmbeddingCacheFile
		if cachePath == "" {
			cachePath = filepath.Join(cfg.DataDir, "embeddings.jsonl")
		}
		// Validate that the cache file stays under DATA_DIR. Path
		// traversal rule: never let user-controllable config write
		// outside the configured data directory.
		absCachePath, err := validateCachePath(cachePath, cfg.DataDir)
		if err != nil {
			slog.Error("embedding cache path rejected", "error", err)
			return
		}

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

		client := NewEmbeddingClient(cfg.GeminiAPIKey, cfg.EmbeddingModel, cfg.EmbeddingBatchSize, cfg.EmbeddingRPM)
		defer client.Close()

		builderCfg := embeddingBuilderConfig{
			Model:     cfg.EmbeddingModel,
			BatchSize: cfg.EmbeddingBatchSize,
		}
		if err := BuildEmbeddingCache(context.Background(), builderCfg, client, server, cache); err != nil {
			slog.Error("failed to build embedding cache", "error", err)
			return
		}
		server.SetEmbeddingCache(cache)

		if cfg.EmbeddingCategorizeEnabled {
			cat := NewCategorizer(Categorizer{
				SimilarityFloor:     cfg.EmbeddingCategorizeSimilarityFloor,
				RecencyWindow:       time.Duration(cfg.EmbeddingCategorizeRecencyWindowDays) * 24 * time.Hour,
				RecentWeight:        cfg.EmbeddingCategorizeRecentWeight,
				PopularityWeight:    cfg.EmbeddingCategorizePopularityWeight,
				MaxSimGate:          cfg.EmbeddingCategorizeMaxSimGate,
				AcceptanceThreshold: cfg.EmbeddingCategorizeAcceptanceThreshold,
			})
			server.SetEmbeddingClient(client)
			server.SetCategorizer(&cat)
			slog.Info("auto_categorize_configured",
				"similarity_floor", cat.SimilarityFloor,
				"recency_window_days", int(cat.RecencyWindow/(24*time.Hour)),
				"recent_weight", cat.RecentWeight,
				"popularity_weight", cat.PopularityWeight,
				"max_sim_gate", cat.MaxSimGate,
				"acceptance_threshold", cat.AcceptanceThreshold,
			)
		} else {
			slog.Info("auto_categorize_disabled_via_config")
		}

		// Suggestion engine — only enabled when embeddings are wired.
		if cfg.SuggestionsEnabled {
			engineCfg := SuggestionEngineConfig{
				MinPurchases:    cfg.SuggestionsMinPurchases,
				MaxInterval:     time.Duration(cfg.SuggestionsMaxIntervalDays) * 24 * time.Hour,
				DueFraction:     cfg.SuggestionsDueFraction,
				DedupSimilarity: cfg.SuggestionsDedupSimilarity,
				RecentLimit:     cfg.SuggestionsRecentLimit,
				RecomputeEvery:  time.Duration(cfg.SuggestionsRecomputeHours) * time.Hour,
			}
			engine := NewSuggestionEngine(engineCfg)
			server.SetSuggestionEngine(engine)
			slog.Info("suggestions_configured",
				"min_purchases", engineCfg.MinPurchases,
				"max_interval_days", cfg.SuggestionsMaxIntervalDays,
				"due_fraction", engineCfg.DueFraction,
				"dedup_similarity", engineCfg.DedupSimilarity,
				"recent_limit", engineCfg.RecentLimit,
				"recompute_interval_hours", cfg.SuggestionsRecomputeHours,
			)
			// Initial recompute now that all events have been loaded
			// (LoadEvents above). Broadcasts no deltas (no clients yet).
			server.RecomputeSuggestions()
			// Periodic recompute. Catches "time-based" transitions
			// (e.g. a previously-fresh item becoming due as time passes).
			go runSuggestionsTicker(server, engine.Config().RecomputeEvery)
		} else {
			slog.Info("suggestions_disabled_via_config")
		}
	} else {
		slog.Info("embedding cache disabled (no GEMINI_API_KEY)")
		slog.Info("suggestions_disabled_no_embeddings")
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
	mux.Handle("/api/v1/auto-categorize/metrics", apiBearerAuth(cfg.APIToken, http.HandlerFunc(server.handleAutoCategorizeMetrics)))
	if cfg.APIToken != "" {
		slog.Info("HTTP API enabled",
			"get_state", "/api/v1/state",
			"post_command", "/api/v1/command",
			"auto_categorize_metrics", "/api/v1/auto-categorize/metrics",
		)
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

// runSuggestionsTicker fires a periodic recompute on the given interval and
// broadcasts any deltas. Runs forever; intended to be launched in its own
// goroutine.
func runSuggestionsTicker(server *Server, interval time.Duration) {
	if interval <= 0 {
		interval = time.Duration(DefaultSuggestionsRecomputeHours) * time.Hour
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		server.RecomputeSuggestions()
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
