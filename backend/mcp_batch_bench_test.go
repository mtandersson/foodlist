package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// BenchmarkRecipeIngredientWrites compares the former per-item command path
// with the recipe batch path for 50 valid ingredients. syncs/op is derived
// from EventStore's measured fsync attempts.
func BenchmarkRecipeIngredientWrites(b *testing.B) {
	for _, tc := range []struct {
		name  string
		batch bool
	}{
		{name: "per_item"},
		{name: "batch", batch: true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			store, err := NewEventStore(filepath.Join(b.TempDir(), "events.jsonl"))
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			srv := NewServer(store)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				commands := make([]CreateTodoCommand, 50)
				for j := range commands {
					commands[j] = CreateTodoCommand{ID: fmt.Sprintf("%d-%d", i, j), Name: fmt.Sprintf("Ingredient %d", j)}
				}
				if tc.batch {
					if n, err := srv.ExecuteCreateTodosBatch(commands); err != nil || n != 50 {
						b.Fatalf("batch: added %d: %v", n, err)
					}
				} else {
					for _, cmd := range commands {
						if err := srv.ExecuteCommand(cmd); err != nil {
							b.Fatal(err)
						}
					}
				}
				b.StopTimer()
				for j := 0; j < 50; j++ {
					<-srv.broadcast
				}
				b.StartTimer()
			}
			b.ReportMetric(float64(store.SyncCount())/float64(b.N), "syncs/op")
		})
	}
}
