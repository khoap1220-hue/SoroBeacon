package store

import (
	"sync"
	"testing"
)

func TestAlertStatsIncrementalTransitions(t *testing.T) {
	tracker := NewAlertStatsTracker()
	ws := "ws-production"

	// 1. Fire 10 alerts
	for i := 0; i < 10; i++ {
		tracker.RecordAlertTransition(ws, "", "firing")
	}

	stats, ok := tracker.GetStats(ws)
	if !ok || stats.FiringAlerts != 10 || stats.TotalAlerts != 10 {
		t.Fatalf("unexpected stats after firing: %+v", stats)
	}

	// 2. Acknowledge 3 alerts
	for i := 0; i < 3; i++ {
		tracker.RecordAlertTransition(ws, "firing", "acknowledged")
	}

	stats, _ = tracker.GetStats(ws)
	if stats.FiringAlerts != 7 || stats.Acknowledged != 3 {
		t.Fatalf("unexpected stats after acknowledge: %+v", stats)
	}

	// 3. Resolve 2 acknowledged alerts and 1 firing alert
	tracker.RecordAlertTransition(ws, "acknowledged", "resolved")
	tracker.RecordAlertTransition(ws, "acknowledged", "resolved")
	tracker.RecordAlertTransition(ws, "firing", "resolved")

	stats, _ = tracker.GetStats(ws)
	if stats.FiringAlerts != 6 || stats.Acknowledged != 1 || stats.ResolvedAlerts != 3 {
		t.Fatalf("unexpected stats after resolve: %+v", stats)
	}
}

func TestAlertStatsConcurrentSafety(t *testing.T) {
	tracker := NewAlertStatsTracker()
	ws := "ws-concurrent"

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				tracker.RecordAlertTransition(ws, "", "firing")
				tracker.RecordAlertTransition(ws, "firing", "resolved")
			}
		}()
	}

	wg.Wait()
	stats, ok := tracker.GetStats(ws)
	if !ok {
		t.Fatal("missing stats")
	}

	expectedTotal := int64(workers * iterations)
	if stats.TotalAlerts != expectedTotal || stats.ResolvedAlerts != expectedTotal || stats.FiringAlerts != 0 {
		t.Errorf("concurrent count mismatch: got %+v, expected total %d", stats, expectedTotal)
	}
}

func TestAlertStatsDriftReconciliation(t *testing.T) {
	tracker := NewAlertStatsTracker()
	ws := "ws-drift"

	tracker.RecordAlertTransition(ws, "", "firing")

	authoritative := PrecomputedAlertStats{
		WorkspaceID:    ws,
		TotalAlerts:    100,
		FiringAlerts:   15,
		Acknowledged:   5,
		ResolvedAlerts: 80,
	}

	tracker.ReconcileDrift(ws, authoritative)

	stats, _ := tracker.GetStats(ws)
	if stats.TotalAlerts != 100 || stats.FiringAlerts != 15 || !stats.IsReconciled {
		t.Errorf("drift reconciliation failed: got %+v", stats)
	}
}
