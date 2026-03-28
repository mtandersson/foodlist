package main

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func apiBearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "MCP HTTP API disabled: set FOODLIST_API_TOKEN", http.StatusServiceUnavailable)
			return
		}
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			http.Error(w, "missing or invalid Authorization", http.StatusUnauthorized)
			return
		}
		got := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if len(got) != len(token) || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAPIState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rollup := StateRollup{
		Type:       "StateRollup",
		Todos:      s.state.GetTodos(),
		Categories: s.state.GetCategories(),
		ListTitle:  s.state.GetListTitle(),
		Version:    version,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rollup); err != nil {
		slog.Error("api state encode failed", "error", err)
	}
}

type apiCommandErrorBody struct {
	Error string `json:"error"`
}

func (s *Server) handleAPICommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	cmd, err := ParseCommand(body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiCommandErrorBody{Error: err.Error()})
		return
	}
	if cmd == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiCommandErrorBody{Error: "unknown command type"})
		return
	}
	if err := s.ExecuteCommand(cmd); err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "persist") || strings.Contains(err.Error(), "marshal") {
			code = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(apiCommandErrorBody{Error: err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(CommandResponse{
		Type:      "CommandResponse",
		CommandID: cmd.GetCommandID(),
		Success:   true,
	})
}
