package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PrecomputedAlertStats maintains incremental counters for alert aggregates
// to avoid expensive full-table scans across partitioned alert tables.
type PrecomputedAlertStats struct {
	WorkspaceID     string    `json:"workspace_id"`
	TotalAlerts     int64     `json:"total_alerts"`
	FiringAlerts    int64     `json:"firing_alerts"`
	Acknowledged    int64     `json:"acknowledged_alerts"`
	ResolvedAlerts  int64     `json:"resolved_alerts"`
	SilencedAlerts  int64     `json:"silenced_alerts"`
	LastUpdated     time.Time `json:"last_updated"`
	IsReconciled    bool      `json:"is_reconciled"`
}

// AlertStatsTracker manages thread-safe incremental alert counters.
type AlertStatsTracker struct {
	mu    sync.RWMutex
	stats map[string]*PrecomputedAlertStats
}

// NewAlertStatsTracker instantiates an in-memory incremental tracker.
func NewAlertStatsTracker() *AlertStatsTracker {
	return &AlertStatsTracker{
		stats: make(map[string]*PrecomputedAlertStats),
	}
}

// RecordAlertTransition atomically moves counts between alert state buckets.
func (t *AlertStatsTracker) RecordAlertTransition(workspaceID, fromState, toState string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.stats[workspaceID]
	if !ok {
		entry = &PrecomputedAlertStats{
			WorkspaceID:  workspaceID,
			LastUpdated:  time.Now().UTC(),
			IsReconciled: true,
		}
		t.stats[workspaceID] = entry
	}

	entry.LastUpdated = time.Now().UTC()

	// Decrement previous state
	switch fromState {
	case "firing":
		if entry.FiringAlerts > 0 {
			entry.FiringAlerts--
		}
	case "acknowledged":
		if entry.Acknowledged > 0 {
			entry.Acknowledged--
		}
	case "silenced":
		if entry.SilencedAlerts > 0 {
			entry.SilencedAlerts--
		}
	}

	// Increment new state
	switch toState {
	case "firing":
		entry.FiringAlerts++
		entry.TotalAlerts++
	case "acknowledged":
		entry.Acknowledged++
	case "resolved":
		entry.ResolvedAlerts++
	case "silenced":
		entry.SilencedAlerts++
	}
}

// GetStats returns the current precomputed stats for a given workspace.
func (t *AlertStatsTracker) GetStats(workspaceID string) (PrecomputedAlertStats, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, ok := t.stats[workspaceID]
	if !ok {
		return PrecomputedAlertStats{WorkspaceID: workspaceID}, false
	}
	return *entry, true
}

// ReconcileDrift compares incremental counters against authoritative counts and corrects drift.
func (t *AlertStatsTracker) ReconcileDrift(workspaceID string, authoritative PrecomputedAlertStats) {
	t.mu.Lock()
	defer t.mu.Unlock()

	authoritative.LastUpdated = time.Now().UTC()
	authoritative.IsReconciled = true
	t.stats[workspaceID] = &authoritative
}
