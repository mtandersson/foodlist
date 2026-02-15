package main

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventStore_FileFormat(t *testing.T) {
	// 1. Setup: Create a temporary file and a new event store.
	tmpfile, err := os.CreateTemp("", "test_events.*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	store, err := NewEventStore(tmpfile.Name())
	require.NoError(t, err)

	// 2. Action: Append a few events to the store.
	events := []Event{
		TodoCreated{ID: "todo-1", Name: "First Task", SortOrder: 1},
		TodoCompleted{ID: "todo-1"},
		CategoryCreated{ID: "cat-1", Name: "Test Category"},
	}

	for _, event := range events {
		err := store.Append(event)
		require.NoError(t, err)
	}
	store.Close() // Close the file to ensure all data is flushed

	// 3. Verification: Read the file and check if each line is a valid JSON object.
	file, err := os.Open(tmpfile.Name())
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var data json.RawMessage
		err := json.Unmarshal(line, &data)
		assert.NoErrorf(t, err, "Line %d should be valid JSON, but got error: %v. Content: %s", lineNum, err, string(line))
	}

	require.NoError(t, scanner.Err(), "Error while scanning the event file")
	assert.Equal(t, len(events), lineNum, "The number of lines should match the number of events appended")
}
