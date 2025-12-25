package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetID_AllEventTypes(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name  string
		event Event
		id    string
	}{
		{
			name: "TodoCreated",
			event: TodoCreated{
				Type:      "TodoCreated",
				ID:        "test-id-1",
				Name:      "Test",
				CreatedAt: now,
				SortOrder: 1000,
			},
			id: "test-id-1",
		},
		{
			name: "TodoCompleted",
			event: TodoCompleted{
				Type:        "TodoCompleted",
				ID:          "test-id-2",
				CompletedAt: now,
			},
			id: "test-id-2",
		},
		{
			name: "TodoUncompleted",
			event: TodoUncompleted{
				Type: "TodoUncompleted",
				ID:   "test-id-3",
			},
			id: "test-id-3",
		},
		{
			name: "TodoStarred",
			event: TodoStarred{
				Type:      "TodoStarred",
				ID:        "test-id-4",
				SortOrder: 5000,
			},
			id: "test-id-4",
		},
		{
			name: "TodoUnstarred",
			event: TodoUnstarred{
				Type: "TodoUnstarred",
				ID:   "test-id-5",
			},
			id: "test-id-5",
		},
		{
			name: "TodoReordered",
			event: TodoReordered{
				Type:      "TodoReordered",
				ID:        "test-id-6",
				SortOrder: 3000,
			},
			id: "test-id-6",
		},
		{
			name: "TodoRenamed",
			event: TodoRenamed{
				Type: "TodoRenamed",
				ID:   "test-id-7",
				Name: "New Name",
			},
			id: "test-id-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.id, tt.event.GetID())
		})
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	_, err := ParseEvent([]byte("not valid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse event type")
}

func TestParseEvent_UnknownEventType(t *testing.T) {
	data := []byte(`{"type":"UnknownEvent","id":"test"}`)
	_, err := ParseEvent(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestParseEvent_InvalidEventData(t *testing.T) {
	// Missing required fields
	data := []byte(`{"type":"TodoCreated","id":"test"}`)
	_, err := ParseEvent(data)
	// Should parse but with zero values for missing fields
	// This is valid JSON unmarshal behavior in Go
	assert.NoError(t, err)
}

func TestParseEvent_MalformedEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		expectErr bool
	}{
		{
			name:      "TodoCompleted malformed",
			data:      `{"type":"TodoCompleted","id":"test","completedAt":"not-a-date"}`,
			expectErr: true,
		},
		{
			name:      "TodoStarred malformed",
			data:      `{"type":"TodoStarred","id":"test","sortOrder":"not-a-number"}`,
			expectErr: true,
		},
		{
			name:      "TodoReordered malformed",
			data:      `{"type":"TodoReordered","id":"test","sortOrder":"not-a-number"}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEvent([]byte(tt.data))
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventType_AllTypes(t *testing.T) {
	now := time.Now().UTC()

	events := []struct {
		event        Event
		expectedType string
	}{
		{TodoCreated{Type: "TodoCreated", ID: "1", Name: "Test", CreatedAt: now, SortOrder: 1000}, "TodoCreated"},
		{TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now}, "TodoCompleted"},
		{TodoUncompleted{Type: "TodoUncompleted", ID: "1"}, "TodoUncompleted"},
		{TodoStarred{Type: "TodoStarred", ID: "1", SortOrder: 5000}, "TodoStarred"},
		{TodoUnstarred{Type: "TodoUnstarred", ID: "1"}, "TodoUnstarred"},
		{TodoReordered{Type: "TodoReordered", ID: "1", SortOrder: 2000}, "TodoReordered"},
		{TodoRenamed{Type: "TodoRenamed", ID: "1", Name: "Renamed"}, "TodoRenamed"},
	}

	for _, e := range events {
		assert.Equal(t, e.expectedType, e.event.EventType())
	}
}

