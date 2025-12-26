package main

import (
	"bufio"
	"fmt"
	"os"
)

// EventStore handles append-only event storage using a JSONL file.
// Concurrency is handled via channels - a single goroutine owns the file.
type EventStore struct {
	filePath string
	file     *os.File
	writeCh  chan writeRequest
	done     chan struct{}
}

type writeRequest struct {
	event    Event
	resultCh chan error
}

// NewEventStore creates a new event store backed by a JSONL file.
// The file is created if it doesn't exist.
func NewEventStore(filePath string) (*EventStore, error) {
	// Open file for appending (create if not exists)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open event store file: %w", err)
	}

	store := &EventStore{
		filePath: filePath,
		file:     file,
		writeCh:  make(chan writeRequest),
		done:     make(chan struct{}),
	}

	// Start the single writer goroutine
	go store.writerLoop()

	return store, nil
}

// writerLoop is the single goroutine that owns file writes.
// All writes go through this goroutine via the writeCh channel.
func (s *EventStore) writerLoop() {
	for {
		select {
		case req := <-s.writeCh:
			err := s.writeEvent(req.event)
			req.resultCh <- err
		case <-s.done:
			return
		}
	}
}

// writeEvent performs the actual write to the file.
// This should only be called from the writerLoop goroutine.
func (s *EventStore) writeEvent(event Event) error {
	data, err := MarshalEvent(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write JSON followed by newline
	_, err = s.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	// Sync to ensure durability
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync event store: %w", err)
	}

	return nil
}

// Append adds an event to the store.
// This is safe to call from multiple goroutines - writes are serialized via channels.
func (s *EventStore) Append(event Event) error {
	resultCh := make(chan error, 1)
	s.writeCh <- writeRequest{event: event, resultCh: resultCh}
	return <-resultCh
}

// ReadAll reads all events from the store.
// This creates a new file handle for reading to avoid interfering with writes.
func (s *EventStore) ReadAll() ([]Event, error) {
	// Open a separate file handle for reading
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open event store for reading: %w", err)
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		event, err := ParseEvent(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse event: %w", err)
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading event store: %w", err)
	}

	return events, nil
}

// Close shuts down the event store, closing the file and stopping the writer goroutine.
func (s *EventStore) Close() error {
	close(s.done)
	return s.file.Close()
}

// AppendRaw appends raw JSON bytes to the store (used when forwarding from WebSocket)
func (s *EventStore) AppendRaw(data []byte) error {
	// Parse to validate
	event, err := ParseEvent(data)
	if err != nil {
		return err
	}
	return s.Append(event)
}
