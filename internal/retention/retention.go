// Package retention provides backup retention policy management.
package retention

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudheed/pgsnap/internal/storage"
)

// Policy defines backup retention rules.
type Policy struct {
	// MaxAge is the maximum age of backups to keep (0 = disabled)
	MaxAge time.Duration

	// MaxCount is the maximum number of backups to keep (0 = disabled)
	MaxCount int

	// KeepDaily is the number of daily backups to keep
	KeepDaily int

	// KeepWeekly is the number of weekly backups to keep
	KeepWeekly int

	// KeepMonthly is the number of monthly backups to keep
	KeepMonthly int
}

// DefaultPolicy returns a default retention policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxAge:      30 * 24 * time.Hour, // 30 days
		MaxCount:    100,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 12,
	}
}

// Result contains the retention operation result.
type Result struct {
	Kept    []string
	Deleted []string
	Errors  []error
}

// Apply applies the retention policy and deletes old backups.
func Apply(ctx context.Context, backend storage.Backend, policy Policy) (*Result, error) {
	result := &Result{}

	// List all backups
	objects, err := backend.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Filter to only backup files
	var backups []storage.ObjectInfo
	for _, obj := range objects {
		if strings.Contains(obj.Key, ".dump") {
			backups = append(backups, obj)
		}
	}

	if len(backups) == 0 {
		return result, nil
	}

	// Sort by date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].LastModified.After(backups[j].LastModified)
	})

	// Determine which backups to keep
	keep := make(map[string]bool)
	now := time.Now()

	// Apply MaxAge
	if policy.MaxAge > 0 {
		cutoff := now.Add(-policy.MaxAge)
		for _, b := range backups {
			if b.LastModified.After(cutoff) {
				keep[b.Key] = true
			}
		}
	} else {
		// If no MaxAge, keep all by default
		for _, b := range backups {
			keep[b.Key] = true
		}
	}

	// Apply MaxCount (keep newest N)
	if policy.MaxCount > 0 && len(backups) > policy.MaxCount {
		// Clear keep map and only keep top N
		keep = make(map[string]bool)
		for i := 0; i < policy.MaxCount && i < len(backups); i++ {
			keep[backups[i].Key] = true
		}
	}

	// Apply KeepDaily, KeepWeekly, KeepMonthly
	if policy.KeepDaily > 0 || policy.KeepWeekly > 0 || policy.KeepMonthly > 0 {
		dailyKept := 0
		weeklyKept := 0
		monthlyKept := 0

		seenDays := make(map[string]bool)
		seenWeeks := make(map[string]bool)
		seenMonths := make(map[string]bool)

		for _, b := range backups {
			dayKey := b.LastModified.Format("2006-01-02")
			year, week := b.LastModified.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", year, week)
			monthKey := b.LastModified.Format("2006-01")

			// Daily
			if policy.KeepDaily > 0 && dailyKept < policy.KeepDaily && !seenDays[dayKey] {
				keep[b.Key] = true
				seenDays[dayKey] = true
				dailyKept++
			}

			// Weekly (keep first backup of each week)
			if policy.KeepWeekly > 0 && weeklyKept < policy.KeepWeekly && !seenWeeks[weekKey] {
				keep[b.Key] = true
				seenWeeks[weekKey] = true
				weeklyKept++
			}

			// Monthly (keep first backup of each month)
			if policy.KeepMonthly > 0 && monthlyKept < policy.KeepMonthly && !seenMonths[monthKey] {
				keep[b.Key] = true
				seenMonths[monthKey] = true
				monthlyKept++
			}
		}
	}

	// Delete backups not in keep set
	for _, b := range backups {
		if keep[b.Key] {
			result.Kept = append(result.Kept, b.Key)
		} else {
			if err := backend.Delete(ctx, b.Key); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to delete %s: %w", b.Key, err))
			} else {
				result.Deleted = append(result.Deleted, b.Key)
			}
		}
	}

	return result, nil
}

// Preview shows what would be deleted without actually deleting.
func Preview(ctx context.Context, backend storage.Backend, policy Policy) (*Result, error) {
	result := &Result{}

	// List all backups
	objects, err := backend.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Filter to only backup files
	var backups []storage.ObjectInfo
	for _, obj := range objects {
		if strings.Contains(obj.Key, ".dump") {
			backups = append(backups, obj)
		}
	}

	if len(backups) == 0 {
		return result, nil
	}

	// Sort by date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].LastModified.After(backups[j].LastModified)
	})

	// Determine which backups to keep (same logic as Apply)
	keep := make(map[string]bool)
	now := time.Now()

	if policy.MaxAge > 0 {
		cutoff := now.Add(-policy.MaxAge)
		for _, b := range backups {
			if b.LastModified.After(cutoff) {
				keep[b.Key] = true
			}
		}
	} else {
		for _, b := range backups {
			keep[b.Key] = true
		}
	}

	if policy.MaxCount > 0 && len(backups) > policy.MaxCount {
		keep = make(map[string]bool)
		for i := 0; i < policy.MaxCount && i < len(backups); i++ {
			keep[backups[i].Key] = true
		}
	}

	if policy.KeepDaily > 0 || policy.KeepWeekly > 0 || policy.KeepMonthly > 0 {
		dailyKept := 0
		weeklyKept := 0
		monthlyKept := 0

		seenDays := make(map[string]bool)
		seenWeeks := make(map[string]bool)
		seenMonths := make(map[string]bool)

		for _, b := range backups {
			dayKey := b.LastModified.Format("2006-01-02")
			year, week := b.LastModified.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", year, week)
			monthKey := b.LastModified.Format("2006-01")

			if policy.KeepDaily > 0 && dailyKept < policy.KeepDaily && !seenDays[dayKey] {
				keep[b.Key] = true
				seenDays[dayKey] = true
				dailyKept++
			}

			if policy.KeepWeekly > 0 && weeklyKept < policy.KeepWeekly && !seenWeeks[weekKey] {
				keep[b.Key] = true
				seenWeeks[weekKey] = true
				weeklyKept++
			}

			if policy.KeepMonthly > 0 && monthlyKept < policy.KeepMonthly && !seenMonths[monthKey] {
				keep[b.Key] = true
				seenMonths[monthKey] = true
				monthlyKept++
			}
		}
	}

	// Categorize results
	for _, b := range backups {
		if keep[b.Key] {
			result.Kept = append(result.Kept, b.Key)
		} else {
			result.Deleted = append(result.Deleted, b.Key)
		}
	}

	return result, nil
}
