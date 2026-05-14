package main

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// autoCategorizeMetrics holds atomic counters for the auto-categorize
// pipeline. All fields are incremented at decision points in
// auto_categorize.go and snapshotted via Snapshot for the metrics endpoint.
// Atomic ints avoid lock contention on the hot path; the read endpoint
// loads each counter independently so a snapshot is eventually consistent
// rather than instantaneously coherent, which is fine for tuning signals.
type autoCategorizeMetrics struct {
	Attempted         atomic.Uint64
	EmbedMiss         atomic.Uint64
	EmbedFailed       atomic.Uint64
	Suggested         atomic.Uint64
	RejectedThreshold atomic.Uint64
	RejectedGate      atomic.Uint64
	RejectedNoSignal  atomic.Uint64
	SkippedUserSet    atomic.Uint64
	SkippedDeleted    atomic.Uint64
	Deduped           atomic.Uint64
	EmitFailed        atomic.Uint64
}

// autoCategorizeSnapshot is the JSON-friendly view of the counters.
type autoCategorizeSnapshot struct {
	Attempted         uint64 `json:"attempted"`
	EmbedMiss         uint64 `json:"embed_miss"`
	EmbedFailed       uint64 `json:"embed_failed"`
	Suggested         uint64 `json:"suggested"`
	RejectedThreshold uint64 `json:"rejected_threshold"`
	RejectedGate      uint64 `json:"rejected_gate"`
	RejectedNoSignal  uint64 `json:"rejected_no_signal"`
	SkippedUserSet    uint64 `json:"skipped_user_set"`
	SkippedDeleted    uint64 `json:"skipped_deleted"`
	Deduped           uint64 `json:"deduped"`
	EmitFailed        uint64 `json:"emit_failed"`
}

// Snapshot returns a point-in-time read of every counter.
func (m *autoCategorizeMetrics) Snapshot() autoCategorizeSnapshot {
	return autoCategorizeSnapshot{
		Attempted:         m.Attempted.Load(),
		EmbedMiss:         m.EmbedMiss.Load(),
		EmbedFailed:       m.EmbedFailed.Load(),
		Suggested:         m.Suggested.Load(),
		RejectedThreshold: m.RejectedThreshold.Load(),
		RejectedGate:      m.RejectedGate.Load(),
		RejectedNoSignal:  m.RejectedNoSignal.Load(),
		SkippedUserSet:    m.SkippedUserSet.Load(),
		SkippedDeleted:    m.SkippedDeleted.Load(),
		Deduped:           m.Deduped.Load(),
		EmitFailed:        m.EmitFailed.Load(),
	}
}

// handleAutoCategorizeMetrics serves a JSON snapshot of the counters.
// Returns 404 when the feature isn't wired (no categorizer attached) so the
// endpoint doesn't reveal feature state to unauthenticated callers — bearer
// auth in main.go already guards it, but layered defaults are cheap.
func (s *Server) handleAutoCategorizeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.categorizer == nil || s.autoCategorizeMetrics == nil {
		http.NotFound(w, r)
		return
	}
	snap := s.autoCategorizeMetrics.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		// Never echo encoder details to clients.
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
