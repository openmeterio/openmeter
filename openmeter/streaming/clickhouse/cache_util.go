package clickhouse

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// isTimeWindowGap checks if there is a gap in the time windows for a given period
func isTimeWindowGap(from time.Time, to time.Time, windowSize meter.WindowSize, rows []meterpkg.MeterQueryRow) bool {
	existingWindows := map[time.Time]time.Time{}
	for _, row := range rows {
		existingWindows[row.WindowStart] = row.WindowEnd
	}

	for from.Before(to) {
		to := from.Add(windowSize.Duration())

		if _, ok := existingWindows[from]; !ok {
			fmt.Println("time window gap", from, to)

			return true
		}

		from = to
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
