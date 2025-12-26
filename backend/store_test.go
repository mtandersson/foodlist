package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventStore_WriteAndRead(t *testing.T) {
	// Create temp file for testing
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	// Create test event
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "test-uuid-1",
		Name:      "Buy milk",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		SortOrder: 1000,
	}

	// Write event
	err = store.Append(event)
	require.NoError(t, err)

	// Read events back
	events, err := store.ReadAll()
	require.NoError(t, err)
	require.Len(t, events, 1)

	// Verify event
	created, ok := events[0].(TodoCreated)
	require.True(t, ok, "expected TodoCreated event")
	assert.Equal(t, event.ID, created.ID)
	assert.Equal(t, event.Name, created.Name)
	assert.Equal(t, event.SortOrder, created.SortOrder)
}

func TestEventStore_MultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UTC()

	// Write multiple events
	events := []Event{
		TodoCreated{
			Type:      "TodoCreated",
			ID:        "uuid-1",
			Name:      "Task 1",
			CreatedAt: now,
			SortOrder: 1000,
		},
		TodoCreated{
			Type:      "TodoCreated",
			ID:        "uuid-2",
			Name:      "Task 2",
			CreatedAt: now,
			SortOrder: 2000,
		},
		TodoCompleted{
			Type:        "TodoCompleted",
			ID:          "uuid-1",
			CompletedAt: now.Add(time.Hour),
		},
	}

	for _, e := range events {
		err = store.Append(e)
		require.NoError(t, err)
	}

	// Read back
	readEvents, err := store.ReadAll()
	require.NoError(t, err)
	assert.Len(t, readEvents, 3)

	// Verify types
	_, ok := readEvents[0].(TodoCreated)
	assert.True(t, ok)
	_, ok = readEvents[1].(TodoCreated)
	assert.True(t, ok)
	_, ok = readEvents[2].(TodoCompleted)
	assert.True(t, ok)
}

func TestEventStore_PersistenceAcrossRestart(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create store and write event
	store1, err := NewEventStore(filePath)
	require.NoError(t, err)

	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "persist-test",
		Name:      "Persistent task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	err = store1.Append(event)
	require.NoError(t, err)
	store1.Close()

	// Create new store instance and verify event persisted
	store2, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store2.Close()

	events, err := store2.ReadAll()
	require.NoError(t, err)
	require.Len(t, events, 1)

	created := events[0].(TodoCreated)
	assert.Equal(t, "persist-test", created.ID)
}

func TestEventStore_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	// Write many events concurrently via channels
	numWriters := 10
	eventsPerWriter := 100
	done := make(chan bool, numWriters)

	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			for j := 0; j < eventsPerWriter; j++ {
				event := TodoCreated{
					Type:      "TodoCreated",
					ID:        "uuid-" + string(rune('A'+writerID)) + "-" + string(rune('0'+j%10)),
					Name:      "Task",
					CreatedAt: time.Now().UTC(),
					SortOrder: writerID*1000 + j,
				}
				store.Append(event)
			}
			done <- true
		}(i)
	}

	// Wait for all writers
	for i := 0; i < numWriters; i++ {
		<-done
	}

	// Verify all events were written
	events, err := store.ReadAll()
	require.NoError(t, err)
	assert.Len(t, events, numWriters*eventsPerWriter)
}

func TestEventStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create empty file
	_, err := os.Create(filePath)
	require.NoError(t, err)

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	events, err := store.ReadAll()
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestEventStore_AllEventTypes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UTC()

	// Write all event types
	allEvents := []Event{
		TodoCreated{Type: "TodoCreated", ID: "1", Name: "Task", CreatedAt: now, SortOrder: 1000},
		TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now},
		TodoUncompleted{Type: "TodoUncompleted", ID: "1"},
		TodoStarred{Type: "TodoStarred", ID: "1", SortOrder: 5000},
		TodoUnstarred{Type: "TodoUnstarred", ID: "1"},
		TodoReordered{Type: "TodoReordered", ID: "1", SortOrder: 2000},
		TodoRenamed{Type: "TodoRenamed", ID: "1", Name: "Renamed Task"},
	}

	for _, e := range allEvents {
		err = store.Append(e)
		require.NoError(t, err)
	}

	// Read back and verify all types
	readEvents, err := store.ReadAll()
	require.NoError(t, err)
	require.Len(t, readEvents, len(allEvents))

	assert.Equal(t, "TodoCreated", readEvents[0].EventType())
	assert.Equal(t, "TodoCompleted", readEvents[1].EventType())
	assert.Equal(t, "TodoUncompleted", readEvents[2].EventType())
	assert.Equal(t, "TodoStarred", readEvents[3].EventType())
	assert.Equal(t, "TodoUnstarred", readEvents[4].EventType())
	assert.Equal(t, "TodoReordered", readEvents[5].EventType())
	assert.Equal(t, "TodoRenamed", readEvents[6].EventType())
}

func TestEventStore_AppendRaw(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UTC()
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "raw-test",
		Name:      "Raw append test",
		CreatedAt: now,
		SortOrder: 1000,
	}

	// Marshal event to JSON
	data, err := MarshalEvent(event)
	require.NoError(t, err)

	// Append raw JSON
	err = store.AppendRaw(data)
	require.NoError(t, err)

	// Read back and verify
	events, err := store.ReadAll()
	require.NoError(t, err)
	require.Len(t, events, 1)

	created := events[0].(TodoCreated)
	assert.Equal(t, "raw-test", created.ID)
	assert.Equal(t, "Raw append test", created.Name)
}

func TestEventStore_AppendRaw_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	// Try to append invalid JSON
	err = store.AppendRaw([]byte("not valid json"))
	assert.Error(t, err)
}

func TestEventStore_ReadAll_SkipEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create file with empty lines
	os.WriteFile(filePath, []byte(`{"type":"TodoCreated","id":"1","name":"Task","createdAt":"2024-01-01T00:00:00Z","sortOrder":1000}

{"type":"TodoCompleted","id":"1","completedAt":"2024-01-01T00:00:00Z"}
`), 0o644)

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	events, err := store.ReadAll()
	require.NoError(t, err)
	// Should skip empty line
	assert.Len(t, events, 2)
}

func TestEventStore_ReadAll_InvalidEvent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create file with invalid event
	os.WriteFile(filePath, []byte(`{"type":"TodoCreated","id":"1","name":"Task","createdAt":"2024-01-01T00:00:00Z","sortOrder":1000}
{"type":"UnknownEvent","id":"2"}
`), 0o644)

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.ReadAll()
	// Should fail on unknown event type
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestEventStore_ReadAll_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create file with malformed JSON
	os.WriteFile(filePath, []byte(`{"type":"TodoCreated","id":"1","name":"Task","createdAt":"2024-01-01T00:00:00Z","sortOrder":1000}
not valid json
`), 0o644)

	store, err := NewEventStore(filePath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.ReadAll()
	// Should fail on malformed JSON
	assert.Error(t, err)
}

func TestEventStore_NewEventStore_InvalidPath(t *testing.T) {
	// Try to create store in non-existent directory without permission
	_, err := NewEventStore("/nonexistent/directory/that/cannot/be/created/events.jsonl")
	assert.Error(t, err)
}
