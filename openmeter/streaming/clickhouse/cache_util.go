package clickhouse

import (
	"math"
	"time"

	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// filterOutNaNValues filters out rows with a value of NaN
// We mark in the cache with NaN if the value is not available
func filterOutNaNValues(rows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	return lo.Filter(rows, func(row meterpkg.MeterQueryRow, _ int) bool {
		return !math.IsNaN(row.Value)
	})
}

// isTimeWindowGap checks if there is a gap in the time windows for a given period
// it only counts as a gap if the gap is in between the existing windows, not at the start or end of the period
func isTimeWindowGap(from time.Time, to time.Time, windowSize meterpkg.WindowSize, rows []meterpkg.MeterQueryRow) bool {
	if len(rows) == 0 {
		return false
	}

	// Create a map of existing windows
	existingWindows := make(map[time.Time]struct{})
	for _, row := range rows {
		existingWindows[row.WindowStart] = struct{}{}
	}

	// Find the first and last window that falls within our from-to range
	var firstWindow, lastWindow *time.Time
	for _, row := range rows {
		if row.WindowStart.Before(from) || row.WindowStart.After(to) {
			continue
		}
		if firstWindow == nil || row.WindowStart.Before(*firstWindow) {
			firstWindow = &row.WindowStart
		}

		// We use the window start window because we care about when the last window started
		if lastWindow == nil || row.WindowStart.After(*lastWindow) {
			lastWindow = &row.WindowStart
		}
	}

	// If we don't have any windows in our range, there's no gap
	if firstWindow == nil || lastWindow == nil {
		return false
	}

	// TODO: refactor me to be more elegant
	wasExisting := false
	existingToNonExisting := false

	// Check for gaps between the first and last window
	current := *firstWindow
	for {
		next := current.Add(windowSize.Duration())

		// Skip if we're at the start or end of the period
		if _, exists := existingWindows[current]; exists {
			wasExisting = true

			// If we're switching pattern again it's a gap
			if existingToNonExisting {
				return true
			}
		} else {
			// Switching from existing to non-existing is a potential gap
			if wasExisting {
				existingToNonExisting = true
			}
		}

		if current.Equal(*lastWindow) {
			break
		}

		current = next
	}

	return false
}

// filterNewRows filters out rows that are not in the cacheable query period
func filterRowsOutOfPeriod(from time.Time, to time.Time, windowSize meterpkg.WindowSize, rows []meterpkg.MeterQueryRow) []meterpkg.MeterQueryRow {
	newRows := []meterpkg.MeterQueryRow{}

	// We filter out rows
	for _, row := range rows {
		// Filter out rows that are before from
		if row.WindowStart.Before(from) {
			continue
		}

		// Filter out rows that are after to
		if row.WindowEnd.After(to) {
			continue
		}

		// Filter out rows that are incomplete windows
		if !row.WindowStart.Truncate(windowSize.Duration()).Equal(row.WindowStart) {
			continue
		}

		newRows = append(newRows, row)
	}

	return newRows
}

// concatAppend concatenates slices of any type
func concatAppend[T any](slices [][]T) []T {
	tmp := []T{}
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}
