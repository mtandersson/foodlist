// Generated from schema/events.schema.json
// Do not edit manually - run schema/generate.sh to regenerate

package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Todo item projected from events
type Todo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt"`
	SortOrder   int        `json:"sortOrder"`
	Starred     bool       `json:"starred"`
}

// Event types
type TodoCreated struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	SortOrder int       `json:"sortOrder"`
}

type TodoCompleted struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	CompletedAt time.Time `json:"completedAt"`
}

type TodoUncompleted struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type TodoStarred struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	SortOrder int    `json:"sortOrder"`
}

type TodoUnstarred struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type TodoReordered struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	SortOrder int    `json:"sortOrder"`
}

type TodoRenamed struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListTitleChanged struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type StateRollup struct {
	Type      string `json:"type"`
	Todos     []Todo `json:"todos"`
	ListTitle string `json:"listTitle"`
}

// Event is an interface for all event types
type Event interface {
	EventType() string
	GetID() string
}

func (e TodoCreated) EventType() string      { return "TodoCreated" }
func (e TodoCompleted) EventType() string    { return "TodoCompleted" }
func (e TodoUncompleted) EventType() string  { return "TodoUncompleted" }
func (e TodoStarred) EventType() string      { return "TodoStarred" }
func (e TodoUnstarred) EventType() string    { return "TodoUnstarred" }
func (e TodoReordered) EventType() string    { return "TodoReordered" }
func (e TodoRenamed) EventType() string      { return "TodoRenamed" }
func (e ListTitleChanged) EventType() string { return "ListTitleChanged" }

func (e TodoCreated) GetID() string      { return e.ID }
func (e TodoCompleted) GetID() string    { return e.ID }
func (e TodoUncompleted) GetID() string  { return e.ID }
func (e TodoStarred) GetID() string      { return e.ID }
func (e TodoUnstarred) GetID() string    { return e.ID }
func (e TodoReordered) GetID() string    { return e.ID }
func (e TodoRenamed) GetID() string      { return e.ID }
func (e ListTitleChanged) GetID() string { return "" } // ListTitleChanged doesn't have an ID

// ParseEvent parses a JSON event into the appropriate Event type
func ParseEvent(data []byte) (Event, error) {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse event type: %w", err)
	}

	switch typeCheck.Type {
	case "TodoCreated":
		var e TodoCreated
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoCreated: %w", err)
		}
		return e, nil
	case "TodoCompleted":
		var e TodoCompleted
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoCompleted: %w", err)
		}
		return e, nil
	case "TodoUncompleted":
		var e TodoUncompleted
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoUncompleted: %w", err)
		}
		return e, nil
	case "TodoStarred":
		var e TodoStarred
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoStarred: %w", err)
		}
		return e, nil
	case "TodoUnstarred":
		var e TodoUnstarred
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoUnstarred: %w", err)
		}
		return e, nil
	case "TodoReordered":
		var e TodoReordered
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoReordered: %w", err)
		}
		return e, nil
	case "TodoRenamed":
		var e TodoRenamed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse TodoRenamed: %w", err)
		}
		return e, nil
	case "ListTitleChanged":
		var e ListTitleChanged
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("failed to parse ListTitleChanged: %w", err)
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", typeCheck.Type)
	}
}

// MarshalEvent serializes an event to JSON
func MarshalEvent(e Event) ([]byte, error) {
	return json.Marshal(e)
}

// ClientCount message sent from server to clients
type ClientCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// AutocompleteRequest is sent by clients to request autocomplete suggestions
type AutocompleteRequest struct {
	Type      string `json:"type"`
	Query     string `json:"query"`
	RequestID string `json:"requestId"`
}

// AutocompleteResponse contains autocomplete suggestions sent back to the requesting client
type AutocompleteResponse struct {
	Type        string   `json:"type"`
	Suggestions []string `json:"suggestions"`
	RequestID   string   `json:"requestId"`
}
