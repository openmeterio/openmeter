package clickhouse

import (
	"time"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

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
		if lastWindow == nil || row.WindowStart.After(*lastWindow) {
			lastWindow = &row.WindowStart
		}
	}

	// If we don't have any windows in our range, there's no gap
	if firstWindow == nil || lastWindow == nil {
		return false
	}

	// Check for gaps between the first and last window
	current := *firstWindow
	for current.Before(*lastWindow) {
		next := current.Add(windowSize.Duration())

		// Skip if we're at the start or end of the period
		if current.Equal(*firstWindow) || next.Equal(*lastWindow) {
			current = next
			continue
		}

		// If the next window doesn't exist, we found a gap
		if _, exists := existingWindows[next]; !exists {
			return true
		}

		current = next
	}

	return false
}

func concatAppend[T any](slices [][]T) []T {
	var tmp []T
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}
