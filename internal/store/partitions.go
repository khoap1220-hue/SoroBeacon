package store

import (
	"context"
	"fmt"
	"time"
)

// PartitionBounds describes start and end bounds for a monthly partition.
type PartitionBounds struct {
	TableName string
	StartTime time.Time
	EndTime   time.Time
}

// ComputeNextMonthlyPartition calculates the partition name and bounds for the upcoming month.
func ComputeNextMonthlyPartition(now time.Time) PartitionBounds {
	year, month, _ := now.Date()
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	afterNext := time.Date(year, month+2, 1, 0, 0, 0, 0, time.UTC)

	return PartitionBounds{
		TableName: fmt.Sprintf("alerts_%04d_%02d", nextMonth.Year(), nextMonth.Month()),
		StartTime: nextMonth,
		EndTime:   afterNext,
	}
}
