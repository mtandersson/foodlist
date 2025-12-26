package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade connection", "error", err)
		return
	}

	client := &Client{
		conn:   conn,
		sendCh: make(chan []byte, 256),
	}

	// Send state rollup to new client BEFORE registering
	// This ensures StateRollup is always the first message
	rollup := StateRollup{
		Type:       "StateRollup",
		Todos:      s.state.GetTodos(),
		Categories: s.state.GetCategories(),
		ListTitle:  s.state.GetListTitle(),
	}
	rollupData, err := json.Marshal(rollup)
	if err != nil {
		slog.Error("failed to marshal state rollup", "error", err)
	} else {
		client.sendCh <- rollupData
	}

	// Start goroutines for reading and writing
	go s.writePump(client)
	go s.readPump(client)

	// Register client last (triggers broadcastClientCount)
	// By this point, writePump is already running and will send StateRollup first
	s.register <- client
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

		// Convert command to event
		event, err := s.commandToEvent(cmd)
		if err != nil {
			slog.Error("failed to convert command", "error", err, "command_type", cmd.GetType(), "commandId", cmd.GetCommandID())
			// Send error response back to the client
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

		// Persist event to store
		if err := s.store.Append(event); err != nil {
			slog.Error("failed to persist event", "error", err, "event_type", event.EventType())
			// Send error response
			response := CommandResponse{
				Type:      "CommandResponse",
				CommandID: cmd.GetCommandID(),
				Success:   false,
				Error:     "failed to persist event",
			}
			if responseData, marshalErr := json.Marshal(response); marshalErr == nil {
				client.sendCh <- responseData
			}
			continue
		}

		// Apply event to state
		s.state.Apply(event)

		// Send success response to the client
		response := CommandResponse{
			Type:      "CommandResponse",
			CommandID: cmd.GetCommandID(),
			Success:   true,
		}
		if responseData, err := json.Marshal(response); err == nil {
			client.sendCh <- responseData
		}

		// Broadcast resulting event to all clients (including sender for confirmation)
		eventData, err := MarshalEvent(event)
		if err != nil {
			slog.Error("failed to marshal event", "error", err, "event_type", event.EventType())
			continue
		}
		s.broadcast <- eventData
	}
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
		return nil, nil
	}
}

// commandToEvent maps incoming commands to domain events
func (s *Server) commandToEvent(cmd Command) (Event, error) {
	switch c := cmd.(type) {
	case CreateTodoCommand:
		// If no ID provided, reject (client should send), but we keep as-is
		if c.ID == "" {
			return nil, nil
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
			Type:       "TodoCreated",
			ID:         c.ID,
			Name:       c.Name,
			CreatedAt:  time.Now().UTC(),
			SortOrder:  sortOrder,
			CategoryID: categoryID,
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

		// Check if an active category with this name already exists
		if s.state.CategoryNameExists(c.Name) {
			return nil, fmt.Errorf("category with name '%s' already exists", c.Name)
		}

		// Check if there's a deleted category with the same name (case-sensitive)
		deletedCategoryID := s.state.FindDeletedCategoryByName(c.Name)

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
			Name:      c.Name,
			CreatedAt: time.Now().UTC(),
			SortOrder: sortOrder,
		}, nil
	case RenameCategoryCommand:
		if _, ok := s.state.GetCategory(c.ID); !ok {
			return nil, fmt.Errorf("category not found")
		}

		// Check if another category with this name already exists
		if s.state.CategoryNameExists(c.Name) {
			// Get the current category to check if it's renaming to itself
			currentCat, _ := s.state.GetCategory(c.ID)
			if currentCat.Name != c.Name {
				return nil, fmt.Errorf("category with name '%s' already exists", c.Name)
			}
		}

		return CategoryRenamed{
			Type: "CategoryRenamed",
			ID:   c.ID,
			Name: c.Name,
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
		return TodoRenamed{
			Type: "TodoRenamed",
			ID:   c.ID,
			Name: c.Name,
		}, nil
	case SetListTitleCommand:
		return ListTitleChanged{
			Type:  "ListTitleChanged",
			Title: c.Title,
		}, nil
	default:
		return nil, nil
	}
}

// LoadEvents loads events from the store and applies them to the state
func (s *Server) LoadEvents() error {
	events, err := s.store.ReadAll()
	if err != nil {
		return err
	}
	s.state.ApplyEvents(events)
	slog.Info("loaded events from store", "event_count", len(events))
	return nil
}
