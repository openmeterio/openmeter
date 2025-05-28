package clickhouse

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/samber/lo"
)

// materializeRows materializes rows for windows that does not have value
func (c *Connector) materializeCacheRows(from time.Time, to time.Time, windowSize meter.WindowSize, rows []meterpkg.MeterQueryRow) ([]meterpkg.MeterQueryRow, error) {
	// Collect group by fields
	groupByFields := []string{}

	if len(rows) > 0 && rows[0].GroupBy != nil {
		for k := range rows[0].GroupBy {
			if !lo.Contains(groupByFields, k) {
				groupByFields = append(groupByFields, k)
			}
		}
	}

	// Collect unique subjects and group bys
	subjectMap := map[*string]struct{}{}
	groupByMap := map[string]map[string]*string{}

	for _, row := range rows {
		if row.Subject != nil {
			if _, ok := subjectMap[row.Subject]; !ok {
				subjectMap[row.Subject] = struct{}{}
			}
		}

		if row.GroupBy != nil {
			groupKey := createGroupKeyFromRow(row, groupByFields)
			if _, ok := groupByMap[groupKey]; !ok {
				// Initialize
				groupByMap[groupKey] = map[string]*string{}

				// Copy
				for k, v := range row.GroupBy {
					groupByMap[groupKey][k] = v
				}
			}
		}
	}

	if len(subjectMap) == 0 {
		subjectMap = map[*string]struct{}{nil: {}}
	}

	// Collect windows for period
	windows := map[time.Time]time.Time{}

	for from.Before(to) {
		to := from.Add(windowSize.Duration())

		windows[from] = to
		from = to
	}

	// Collect existing rows
	existingRows := map[string]struct{}{}
	for _, row := range rows {
		key := createKeyFromRow(row, &windowSize, groupByFields)
		existingRows[key] = struct{}{}
	}

	// Materialize missing rows
	materializedRows := []meterpkg.MeterQueryRow{}

	for windowStart, windowEnd := range windows {
		for subject := range subjectMap {
			if len(groupByMap) == 0 {
				materializedRow := meterpkg.MeterQueryRow{
					WindowStart: windowStart,
					WindowEnd:   windowEnd,
					Subject:     subject,
				}

				key := createKeyFromRow(materializedRow, &windowSize, groupByFields)

				if _, ok := existingRows[key]; !ok {
					materializedRows = append(materializedRows, materializedRow)
				}
			} else {
				for _, groupBy := range groupByMap {
					materializedRow := meterpkg.MeterQueryRow{
						WindowStart: windowStart,
						WindowEnd:   windowEnd,
						Subject:     subject,
						GroupBy:     groupBy,
					}

					key := createKeyFromRow(materializedRow, &windowSize, groupByFields)

					if _, ok := existingRows[key]; !ok {
						materializedRows = append(materializedRows, materializedRow)
					}
				}
			}
		}
	}

	return materializedRows, nil
}
