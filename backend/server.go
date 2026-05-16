package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/singleflight"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	sendCh chan []byte
}

// Server manages WebSocket connections and event broadcasting
type Server struct {
	store      *EventStore
	state      *State
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	// embeddingCache, if set, holds the in-memory embedding cache populated
	// at startup. Used by the auto-categorize feature.
	embeddingCache *EmbeddingCache

	// Auto-categorize dependencies. All four fields must be set for the
	// feature to activate; any nil disables it as a no-op.
	embeddingClient       Embedder
	categorizer           *Categorizer
	suggestFlight         *singleflight.Group
	autoCategorizeMetrics *autoCategorizeMetrics

	// suggestions is the optional engine that computes "things you should
	// probably buy soon" hints. Requires the embedding stack; nil when
	// disabled. Access is safe (nil-checked) from event hooks.
	suggestions *SuggestionEngine
}

// ClientCountMessage informs clients of current connected user count
type ClientCountMessage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// CommandResponse is sent to a client in response to a command
type CommandResponse struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// Command represents an incoming action from the client
type Command interface {
	GetType() string
	GetCommandID() string
}

type BaseCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
}

func (c BaseCommand) GetType() string      { return c.Type }
func (c BaseCommand) GetCommandID() string { return c.CommandID }

type CreateTodoCommand struct {
	BaseCommand
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	SortOrder  float64 `json:"sortOrder,omitempty"`
	CategoryID *string `json:"categoryId,omitempty"`
}

type CategorizeTodoCommand struct {
	BaseCommand
	ID         string  `json:"id"`
	CategoryID *string `json:"categoryId"`
}

type CreateCategoryCommand struct {
	BaseCommand
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	SortOrder float64 `json:"sortOrder,omitempty"`
}

type RenameCategoryCommand struct {
	BaseCommand
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeleteCategoryCommand struct {
	BaseCommand
	ID string `json:"id"`
}

type ReorderCategoryCommand struct {
	BaseCommand
	ID        string  `json:"id"`
	SortOrder float64 `json:"sortOrder"`
}

type CompleteTodoCommand struct {
	BaseCommand
	ID string `json:"id"`
}

type UncompleteTodoCommand struct {
	BaseCommand
	ID string `json:"id"`
}

type StarTodoCommand struct {
	BaseCommand
	ID string `json:"id"`
}

type UnstarTodoCommand struct {
	BaseCommand
	ID string `json:"id"`
}

type ReorderTodoCommand struct {
	BaseCommand
	ID        string  `json:"id"`
	SortOrder float64 `json:"sortOrder"`
}

type RenameTodoCommand struct {
	BaseCommand
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SetListTitleCommand struct {
	BaseCommand
	Title string `json:"title"`
}

// NewServer creates a new WebSocket server
func NewServer(store *EventStore) *Server {
	return &Server{
		store:      store,
		state:      NewState(),
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// SetEmbeddingCache attaches a populated embedding cache to the server.
// Intended to be called once at startup before Run.
func (s *Server) SetEmbeddingCache(c *EmbeddingCache) {
	s.embeddingCache = c
}

// EmbeddingCache returns the attached embedding cache, or nil if none is set.
func (s *Server) EmbeddingCache() *EmbeddingCache {
	return s.embeddingCache
}

// SetEmbeddingClient attaches the embedder used by auto-categorize to fetch
// on cache miss. Pass nil to leave the feature disabled.
func (s *Server) SetEmbeddingClient(e Embedder) {
	s.embeddingClient = e
	if e != nil && s.suggestFlight == nil {
		s.suggestFlight = &singleflight.Group{}
	}
	if e != nil && s.autoCategorizeMetrics == nil {
		s.autoCategorizeMetrics = &autoCategorizeMetrics{}
	}
}

// SetCategorizer attaches the pure scorer used by auto-categorize. Pass nil
// to leave the feature disabled.
func (s *Server) SetCategorizer(c *Categorizer) {
	s.categorizer = c
	if c != nil && s.suggestFlight == nil {
		s.suggestFlight = &singleflight.Group{}
	}
	if c != nil && s.autoCategorizeMetrics == nil {
		s.autoCategorizeMetrics = &autoCategorizeMetrics{}
	}
}

// AutoCategorizeEnabled reports whether every dependency is wired.
func (s *Server) AutoCategorizeEnabled() bool {
	return s.embeddingCache != nil && s.embeddingClient != nil &&
		s.categorizer != nil && s.suggestFlight != nil &&
		s.autoCategorizeMetrics != nil
}

// SetSuggestionEngine attaches an engine. Pass nil to disable the feature.
func (s *Server) SetSuggestionEngine(e *SuggestionEngine) {
	s.suggestions = e
}

// SuggestionsEnabled reports whether the suggestion engine is configured.
func (s *Server) SuggestionsEnabled() bool {
	return s.suggestions != nil && s.embeddingCache != nil
}

// RecomputeSuggestions runs a full recompute and broadcasts the resulting
// deltas (or rollup for large changes). Safe to call when the engine is
// disabled (no-op). Used at startup and from the periodic ticker.
func (s *Server) RecomputeSuggestions() {
	if !s.SuggestionsEnabled() {
		return
	}
	todos := s.state.GetTodos()
	cats := s.state.GetCategories()
	embs := s.embeddingCache.All()
	added, removed := s.suggestions.Recompute(todos, embs, cats)
	s.broadcastSuggestionDeltas(added, removed)
}

// broadcastSuggestionDeltas serializes and queues SuggestionAdded /
// SuggestionRemoved messages for every connected client. Uses non-blocking
// sends so a slow broadcast loop (or a full channel) drops the delta rather
// than blocking the caller — the next periodic recompute (or reconnect)
// will reconcile state.
func (s *Server) broadcastSuggestionDeltas(added []Suggestion, removed []string) {
	for _, id := range removed {
		msg := SuggestionRemoved{Type: "SuggestionRemoved", ID: id}
		s.enqueueBroadcast(msg, "SuggestionRemoved")
	}
	for _, sg := range added {
		msg := SuggestionAdded{Type: "SuggestionAdded", Suggestion: sg}
		s.enqueueBroadcast(msg, "SuggestionAdded")
	}
}

// enqueueBroadcast marshals msg and pushes it onto s.broadcast without
// blocking. On marshal error or a full channel it logs (without exposing
// message contents) and returns.
func (s *Server) enqueueBroadcast(msg any, label string) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal broadcast message", "label", label, "error", err)
		return
	}
	select {
	case s.broadcast <- data:
	default:
		slog.Warn("broadcast channel full, dropping message", "label", label)
	}
}

// maybeUpdateSuggestionsForEvent reacts to events that may affect the
// suggestion set. Runs the recompute synchronously (it's cheap: pure
// in-memory) but after the originating event has been broadcast.
//
// Optimistic short-circuit: when the event is a TodoCreated, immediately
// remove any matching suggestion so the UI feels instant; the full
// recompute then reconciles canonical state.
//
// The switch enumerates every event class whose data flows into
// SuggestionEngine.computeSuggestions: completion timestamps, names,
// category assignments, and the live category set.
func (s *Server) maybeUpdateSuggestionsForEvent(event Event) {
	if !s.SuggestionsEnabled() {
		return
	}
	switch ev := event.(type) {
	case TodoCreated:
		if id, ok := s.suggestions.MarkPurchased(ev.Name); ok {
			s.broadcastSuggestionDeltas(nil, []string{id})
		}
	case TodoCompleted,
		TodoUncompleted,
		TodoRenamed,
		TodoCategorized,
		CategoryCreated,
		CategoryDeleted,
		CategoryRenamed:
		// fall through to full recompute below
	default:
		return
	}
	s.RecomputeSuggestions()
}

// Run starts the server's main event loop
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
			slog.Info("client connected", "total_clients", len(s.clients))
			s.broadcastClientCount()

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.sendCh)
				slog.Info("client disconnected", "total_clients", len(s.clients))
				s.broadcastClientCount()
			}

		case message := <-s.broadcast:
			for client := range s.clients {
				select {
				case client.sendCh <- message:
				default:
					// Client's send buffer is full, disconnect
					close(client.sendCh)
					delete(s.clients, client)
				}
			}
		}
	}
}

// HandleWebSocket handles WebSocket upgrade and client communication
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract client IP and proxy headers
	clientIP := r.RemoteAddr
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	xRealIP := r.Header.Get("X-Real-IP")

	// Log new WebSocket connection with IP and proxy headers
	logAttrs := []any{
		"remote_addr", clientIP,
	}
	if xForwardedFor != "" {
		logAttrs = append(logAttrs, "x_forwarded_for", xForwardedFor)
	}
	if xRealIP != "" {
		logAttrs = append(logAttrs, "x_real_ip", xRealIP)
	}
	slog.Info("new websocket connection", logAttrs...)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade connection", "error", err)
		return
	}

	client := &Client{
		conn:   conn,
		sendCh: make(chan []byte, 256),
	}

	s.register <- client

	// Send state rollup to new client
	rollup := StateRollup{
		Type:       "StateRollup",
		Todos:      s.state.GetTodos(),
		Categories: s.state.GetCategories(),
		ListTitle:  s.state.GetListTitle(),
		Version:    version,
	}
	if s.SuggestionsEnabled() {
		rollup.FeatureFlags = &FeatureFlags{Suggestions: true}
	}
	rollupData, err := json.Marshal(rollup)
	if err != nil {
		slog.Error("failed to marshal state rollup", "error", err)
	} else {
		client.sendCh <- rollupData
	}

	// Send suggestion snapshot to the new client (not broadcast) when
	// the engine is enabled. This is the initial "array replace" message.
	if s.SuggestionsEnabled() {
		snapshot := SuggestionsRollup{
			Type:        "SuggestionsRollup",
			Suggestions: s.suggestions.Snapshot(),
		}
		if data, err := json.Marshal(snapshot); err != nil {
			slog.Error("failed to marshal suggestions rollup", "error", err)
		} else {
			select {
			case client.sendCh <- data:
			default:
				slog.Warn("client send buffer full, dropping suggestions rollup")
			}
		}
	}

	// Start goroutines for reading and writing
	go s.writePump(client)
	go s.readPump(client)
}

// broadcastClientCount sends the current number of connected clients to all clients
func (s *Server) broadcastClientCount() {
	msg := ClientCountMessage{
		Type:  "ClientCount",
		Count: len(s.clients),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal client count", "error", err)
		return
	}
	s.broadcast <- data
}

// writePump sends messages from the send channel to the WebSocket
func (s *Server) writePump(client *Client) {
	defer func() {
		client.conn.Close()
	}()

	for message := range client.sendCh {
		err := client.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			slog.Error("error writing message", "error", err)
			return
		}
	}
}

// readPump reads messages from the WebSocket and processes events
func (s *Server) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket error", "error", err)
			}
			return
		}

		// Check if this is an autocomplete request first
		if handled := s.handleAutocompleteRequest(client, message); handled {
			continue
		}

		// Parse and validate command
		cmd, err := ParseCommand(message)
		if err != nil {
			slog.Warn("invalid command received", "error", err, "message", string(message))
			continue
		}
		if cmd == nil {
			slog.Warn("unknown command received", "message", string(message))
			continue
		}

		// Log received command
		slog.Info("command received", "type", cmd.GetType(), "commandId", cmd.GetCommandID(), "message", string(message))

		event, eventData, err := s.applyCommand(cmd)
		if err != nil {
			slog.Error("command failed", "error", err, "command_type", cmd.GetType(), "commandId", cmd.GetCommandID())
			response := CommandResponse{
				Type:      "CommandResponse",
				CommandID: cmd.GetCommandID(),
				Success:   false,
				Error:     err.Error(),
			}
			if responseData, marshalErr := json.Marshal(response); marshalErr == nil {
				client.sendCh <- responseData
			}
			continue
		}

		// Match historical ordering: acknowledge the sender before broadcasting the event
		// to all clients (including the sender), so integration tests and UIs see CommandResponse first.
		response := CommandResponse{
			Type:      "CommandResponse",
			CommandID: cmd.GetCommandID(),
			Success:   true,
		}
		if responseData, err := json.Marshal(response); err == nil {
			client.sendCh <- responseData
		}
		s.broadcast <- eventData
		// Fire auto-categorize AFTER broadcast so any follow-up
		// TodoCategorized strictly trails the TodoCreated on the wire.
		s.maybeStartAutoCategorize(event)
		// Update suggestion engine after the originating event has
		// already been queued for broadcast.
		s.maybeUpdateSuggestionsForEvent(event)
	}
}

// applyCommand converts cmd to an event, persists it, applies it to state, and returns the
// produced event plus its wire-format bytes for broadcasting. It does not
// broadcast or send CommandResponse.
func (s *Server) applyCommand(cmd Command) (Event, []byte, error) {
	if cmd == nil {
		return nil, nil, fmt.Errorf("nil command")
	}
	event, err := s.commandToEvent(cmd)
	if err != nil {
		return nil, nil, err
	}
	if event == nil {
		return nil, nil, fmt.Errorf("invalid command")
	}
	if err := s.store.Append(event); err != nil {
		return nil, nil, fmt.Errorf("failed to persist event: %w", err)
	}
	s.state.ApplyEvents([]Event{event})
	eventData, err := MarshalEvent(event)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return event, eventData, nil
}

// ExecuteCommand validates the command, persists the resulting event, updates in-memory state,
// and broadcasts the event to all WebSocket clients.
//
// The auto-categorize hook (if configured) is dispatched AFTER the broadcast
// so any follow-up TodoCategorized always arrives strictly after the
// originating TodoCreated on the wire.
func (s *Server) ExecuteCommand(cmd Command) error {
	event, eventData, err := s.applyCommand(cmd)
	if err != nil {
		return err
	}
	s.broadcast <- eventData
	s.maybeStartAutoCategorize(event)
	s.maybeUpdateSuggestionsForEvent(event)
	return nil
}

// handleAutocompleteRequest checks if the message is an autocomplete request and handles it
// Returns true if the message was an autocomplete request, false otherwise
func (s *Server) handleAutocompleteRequest(client *Client, message []byte) bool {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &typeCheck); err != nil {
		return false
	}

	if typeCheck.Type != "AutocompleteRequest" {
		return false
	}

	var req AutocompleteRequest
	if err := json.Unmarshal(message, &req); err != nil {
		slog.Warn("failed to parse autocomplete request", "error", err)
		return true
	}

	// Get suggestions
	suggestions := s.getAutocompleteSuggestions(req.Query)

	// Create response
	response := AutocompleteResponse{
		Type:        "AutocompleteResponse",
		Suggestions: suggestions,
		RequestID:   req.RequestID,
	}

	// Send response only to the requesting client
	responseData, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to marshal autocomplete response", "error", err)
		return true
	}

	// Send directly to client, not broadcast
	select {
	case client.sendCh <- responseData:
	default:
		slog.Warn("client send buffer full, dropping autocomplete response")
	}

	return true
}

// ParseCommand unmarshals incoming JSON into the correct command type
func ParseCommand(data []byte) (Command, error) {
	var base BaseCommand
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case "CreateTodo":
		var cmd CreateTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "CategorizeTodo":
		var cmd CategorizeTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "CreateCategory":
		var cmd CreateCategoryCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "RenameCategory":
		var cmd RenameCategoryCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "DeleteCategory":
		var cmd DeleteCategoryCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "ReorderCategory":
		var cmd ReorderCategoryCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "CompleteTodo":
		var cmd CompleteTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "UncompleteTodo":
		var cmd UncompleteTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "StarTodo":
		var cmd StarTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "UnstarTodo":
		var cmd UnstarTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "ReorderTodo":
		var cmd ReorderTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "RenameTodo":
		var cmd RenameTodoCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "SetListTitle":
		var cmd SetListTitleCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	default:
		return nil, fmt.Errorf("unknown command type: %q", base.Type)
	}
}

// commandToEvent maps incoming commands to domain events
func (s *Server) commandToEvent(cmd Command) (Event, error) {
	switch c := cmd.(type) {
	case CreateTodoCommand:
		if c.ID == "" {
			return nil, fmt.Errorf("missing todo id")
		}
		// Trim whitespace from name
		trimmedName := strings.TrimSpace(c.Name)
		if trimmedName == "" {
			return nil, fmt.Errorf("todo name cannot be empty")
		}
		parsed := ParseIngredientInput(trimmedName)
		name := parsed.Name
		if name == "" {
			name = trimmedName
		}
		var categoryID *string
		if c.CategoryID != nil {
			categoryID = c.CategoryID
		}
		sortOrder := s.state.GetHighestSortOrder() + 1000
		if c.SortOrder != 0 {
			sortOrder = int(c.SortOrder)
		}
		return TodoCreated{
			Type:          "TodoCreated",
			ID:            c.ID,
			Name:          name,
			CreatedAt:     time.Now().UTC(),
			SortOrder:     sortOrder,
			CategoryID:    categoryID,
			Count:         parsed.Count,
			Unit:          parsed.Unit,
			OriginalInput: parsed.OriginalInput,
		}, nil
	case CategorizeTodoCommand:
		// Validate category exists if provided
		if c.CategoryID != nil {
			if _, ok := s.state.GetCategory(*c.CategoryID); !ok {
				return nil, fmt.Errorf("category does not exist")
			}
		}
		if _, ok := s.state.GetTodo(c.ID); !ok {
			return nil, fmt.Errorf("todo not found")
		}
		return TodoCategorized{
			Type:       "TodoCategorized",
			ID:         c.ID,
			CategoryID: c.CategoryID,
		}, nil
	case CreateCategoryCommand:
		if c.ID == "" {
			return nil, fmt.Errorf("missing category id")
		}

		// Trim whitespace from name
		trimmedName := strings.TrimSpace(c.Name)
		if trimmedName == "" {
			return nil, fmt.Errorf("category name cannot be empty")
		}

		// Check if an active category with this name already exists
		if s.state.CategoryNameExists(trimmedName) {
			return nil, fmt.Errorf("category with name '%s' already exists", trimmedName)
		}

		// Check if there's a deleted category with the same name (case-sensitive)
		deletedCategoryID := s.state.FindDeletedCategoryByName(trimmedName)

		// If a deleted category with this name exists, reuse its ID
		categoryID := c.ID
		if deletedCategoryID != "" {
			categoryID = deletedCategoryID
		}

		sortOrder := s.state.GetHighestCategorySortOrder() + 1000
		if c.SortOrder != 0 {
			sortOrder = int(c.SortOrder)
		}
		return CategoryCreated{
			Type:      "CategoryCreated",
			ID:        categoryID,
			Name:      trimmedName,
			CreatedAt: time.Now().UTC(),
			SortOrder: sortOrder,
		}, nil
	case RenameCategoryCommand:
		if _, ok := s.state.GetCategory(c.ID); !ok {
			return nil, fmt.Errorf("category not found")
		}

		// Trim whitespace from name
		trimmedName := strings.TrimSpace(c.Name)
		if trimmedName == "" {
			return nil, fmt.Errorf("category name cannot be empty")
		}

		// Check if another category with this name already exists
		if s.state.CategoryNameExists(trimmedName) {
			// Get the current category to check if it's renaming to itself
			currentCat, _ := s.state.GetCategory(c.ID)
			if currentCat.Name != trimmedName {
				return nil, fmt.Errorf("category with name '%s' already exists", trimmedName)
			}
		}

		return CategoryRenamed{
			Type: "CategoryRenamed",
			ID:   c.ID,
			Name: trimmedName,
		}, nil
	case DeleteCategoryCommand:
		if s.state.CategoryHasTodos(c.ID) {
			return nil, fmt.Errorf("cannot delete non-empty category")
		}
		if _, ok := s.state.GetCategory(c.ID); !ok {
			return nil, fmt.Errorf("category not found")
		}
		return CategoryDeleted{
			Type: "CategoryDeleted",
			ID:   c.ID,
		}, nil
	case ReorderCategoryCommand:
		if _, ok := s.state.GetCategory(c.ID); !ok {
			return nil, fmt.Errorf("category not found")
		}
		return CategoryReordered{
			Type:      "CategoryReordered",
			ID:        c.ID,
			SortOrder: int(c.SortOrder),
		}, nil
	case CompleteTodoCommand:
		return TodoCompleted{
			Type:        "TodoCompleted",
			ID:          c.ID,
			CompletedAt: time.Now().UTC(),
		}, nil
	case UncompleteTodoCommand:
		return TodoUncompleted{
			Type: "TodoUncompleted",
			ID:   c.ID,
		}, nil
	case StarTodoCommand:
		return TodoStarred{
			Type:      "TodoStarred",
			ID:        c.ID,
			SortOrder: s.state.GetHighestSortOrder() + 1000,
		}, nil
	case UnstarTodoCommand:
		return TodoUnstarred{
			Type: "TodoUnstarred",
			ID:   c.ID,
		}, nil
	case ReorderTodoCommand:
		return TodoReordered{
			Type:      "TodoReordered",
			ID:        c.ID,
			SortOrder: int(c.SortOrder),
		}, nil
	case RenameTodoCommand:
		// Trim whitespace from name
		trimmedName := strings.TrimSpace(c.Name)
		if trimmedName == "" {
			return nil, fmt.Errorf("todo name cannot be empty")
		}
		parsed := ParseIngredientInput(trimmedName)
		name := parsed.Name
		if name == "" {
			name = trimmedName
		}
		return TodoRenamed{
			Type:          "TodoRenamed",
			ID:            c.ID,
			Name:          name,
			Count:         parsed.Count,
			Unit:          parsed.Unit,
			OriginalInput: parsed.OriginalInput,
		}, nil
	case SetListTitleCommand:
		// Trim whitespace from title
		trimmedTitle := strings.TrimSpace(c.Title)
		return ListTitleChanged{
			Type:  "ListTitleChanged",
			Title: trimmedTitle,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported command")
	}
}

// LoadEvents loads events from the store and applies them to the state
func (s *Server) LoadEvents() error {
	events, err := s.store.ReadAll()
	if err != nil {
		return err
	}
	s.state.ApplyEvents(events)
	s.rebuildAutocomplete()
	slog.Info("loaded events from store", "event_count", len(events))
	return nil
}

// rebuildAutocomplete rebuilds the entire autocomplete index from the event history.
func (s *Server) rebuildAutocomplete() {
	s.state.autocomplete.Reset()
	events, err := s.store.ReadAll()
	if err != nil {
		slog.Error("failed to read all events for autocomplete rebuild", "error", err)
		return
	}
	for _, event := range events {
		var todo *Todo
		if e, ok := event.(interface{ GetID() string }); ok {
			todo = s.state.todos[e.GetID()]
		}
		s.state.autocomplete.Apply(event, todo)
	}
}
