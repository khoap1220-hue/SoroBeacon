package store

import (
	"testing"
	"time"
)

func TestComputeNextMonthlyPartition(t *testing.T) {
	sampleDate := time.Date(2026, time.November, 4, 12, 0, 0, 0, time.UTC)
	bounds := ComputeNextMonthlyPartition(sampleDate)

	expectedTable := "alerts_2026_12"
	if bounds.TableName != expectedTable {
		t.Errorf("got table %s, want %s", bounds.TableName, expectedTable)
	}

	expectedStart := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	if !bounds.StartTime.Equal(expectedStart) {
		t.Errorf("got start %v, want %v", bounds.StartTime, expectedStart)
	}
}
