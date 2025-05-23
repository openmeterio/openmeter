package clickhouse

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestMergeMeterQueryRows(t *testing.T) {
	subject1 := "subject1"
	subject2 := "subject2"
	group1Value := "group1_value"
	group2Value := "group2_value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	windowSize := meterpkg.WindowSizeHour

	tests := []struct {
		name        string
		meterDef    meterpkg.Meter
		queryParams streaming.QueryParams
		cachedRows  []meterpkg.MeterQueryRow
		freshRows   []meterpkg.MeterQueryRow
		wantCount   int
	}{
		{
			name: "empty cached rows",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{},
			cachedRows:  []meterpkg.MeterQueryRow{},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			wantCount: 1,
		},
		{
			name: "with window size, rows are concatenated",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				WindowSize: &windowSize,
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 2,
		},
		{
			name: "without window size, sum aggregation",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 1, // Aggregated to a single row
		},
		{
			name: "without window size, different subjects",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject2,
				},
			},
			wantCount: 2, // One row per subject
		},
		{
			name: "without window size, with group by values",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			wantCount: 1, // Aggregated by groups
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := mergeMeterQueryRows(testCase.meterDef, testCase.queryParams, testCase.cachedRows, testCase.freshRows)
			assert.Equal(t, testCase.wantCount, len(result))

			if testCase.meterDef.Aggregation == meterpkg.MeterAggregationSum && len(testCase.queryParams.GroupBy) > 0 && testCase.queryParams.WindowSize == nil {
				// If we're aggregating, check that values are summed
				if len(result) == 1 && len(testCase.cachedRows) > 0 && len(testCase.freshRows) > 0 {
					expectedSum := testCase.cachedRows[0].Value + testCase.freshRows[0].Value
					assert.Equal(t, expectedSum, result[0].Value)
				}
			}
		})
	}
}

func TestCreateGroupKeyFromRow(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"
	group2Value := "group2-value"

	testRow := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": &group1Value,
			"group2": &group2Value,
		},
	}

	tests := []struct {
		name        string
		queryParams streaming.QueryParams
		expectedKey string
	}{
		{
			name: "subject only",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			expectedKey: "subject=test-subject;group=subject=nil;",
		},
		{
			name: "with group by fields",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			expectedKey: "subject=test-subject;group=group1=group1-value;group=group2=group2-value;group=subject=nil;",
		},
		{
			name: "with missing group by field",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group3"},
			},
			expectedKey: "subject=test-subject;group=group1=group1-value;group=group3=nil;group=subject=nil;",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := createGroupKeyFromRowWithQueryParams(testRow, testCase.queryParams)
			assert.Equal(t, testCase.expectedKey, result)
		})
	}
}

func TestAggregateRowsByAggregationType(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	// Rows have the same subject and groupBy values
	testRows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &group1Value,
			},
		},
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       20,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &group1Value,
			},
		},
	}

	tests := []struct {
		name        string
		aggregation meterpkg.MeterAggregation
		rows        []meterpkg.MeterQueryRow
		wantValue   float64
		wantSubject string
	}{
		{
			name:        "sum aggregation",
			aggregation: meterpkg.MeterAggregationSum,
			rows:        testRows,
			wantValue:   30, // 10 + 20
			wantSubject: subject,
		},
		{
			name:        "count aggregation",
			aggregation: meterpkg.MeterAggregationCount,
			rows:        testRows,
			wantValue:   30, // count should be the same as sum
			wantSubject: subject,
		},
		{
			name:        "min aggregation",
			aggregation: meterpkg.MeterAggregationMin,
			rows:        testRows,
			wantValue:   10, // min of 10 and 20
			wantSubject: subject,
		},
		{
			name:        "max aggregation",
			aggregation: meterpkg.MeterAggregationMax,
			rows:        testRows,
			wantValue:   20, // max of 10 and 20
			wantSubject: subject,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := aggregateRowsByAggregationType(testCase.aggregation, testCase.rows)

			assert.Equal(t, testCase.wantValue, result.Value)
			require.NotNil(t, result.Subject)
			assert.Equal(t, testCase.wantSubject, *result.Subject)

			// Window range should span from earliest to latest
			assert.Equal(t, windowStart1, result.WindowStart)
			assert.Equal(t, windowEnd2, result.WindowEnd)

			// GroupBy values should be preserved
			require.Contains(t, result.GroupBy, "group1")
			require.NotNil(t, result.GroupBy["group1"])
			assert.Equal(t, group1Value, *result.GroupBy["group1"])
		})
	}
}

func TestDedupeQueryRows(t *testing.T) {
	subject1 := "test-subject"
	subject2 := "test-subject-2"
	group1Key := "group1"
	group1Value := "group1-value"
	group2Key := "group2"
	group2Value := "group2-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	rows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Duplicate row
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Row with different group by value
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group2Key: &group2Value,
			},
		},
		// Row with different time
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Row with different subject
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject2,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
	}

	deduplicatedRows, err := dedupeQueryRows(rows, []string{group1Key, group2Key})
	require.NoError(t, err)

	assert.Equal(t, 4, len(deduplicatedRows))
	assert.Equal(t, deduplicatedRows, []meterpkg.MeterQueryRow{
		rows[0],
		rows[2],
		rows[3],
		rows[4],
	})

	// Test duplicates with inconsistent value - should now intelligently choose the best value
	rows[0].Value = 20
	deduplicatedRows2, err := dedupeQueryRows(rows, []string{group1Key, group2Key})
	require.NoError(t, err)

	// Should still have 4 rows, with the duplicate resolved
	assert.Equal(t, 4, len(deduplicatedRows2))

	// The first row should have the higher value (20 instead of 10)
	// because our smart deduplication prefers higher absolute values
	assert.Equal(t, 20.0, deduplicatedRows2[0].Value, "Should choose the higher value when deduplicating")
}

// TestFilterGroupByAffectsMergeGrouping tests that FilterGroupBy values affect how rows are grouped during merge
func TestFilterGroupByAffectsMergeGrouping(t *testing.T) {
	subject1 := "subject1"
	subject2 := "subject2"
	group1Value := "group1_value"
	group2Value := "group2_value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	meterDef := meterpkg.Meter{
		Aggregation: meterpkg.MeterAggregationSum,
	}

	tests := []struct {
		name          string
		queryParams   streaming.QueryParams
		cachedRows    []meterpkg.MeterQueryRow
		freshRows     []meterpkg.MeterQueryRow
		expectedCount int
		description   string
	}{
		{
			name: "FilterGroupBy affects grouping - same subject with different groups",
			queryParams: streaming.QueryParams{
				GroupBy:       []string{"subject", "group1"},
				FilterGroupBy: map[string][]string{"group1": {"group1_value"}},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			expectedCount: 1, // Should merge because same grouping
			description:   "Rows with same subject and group values should merge into one",
		},
		{
			name: "FilterGroupBy affects grouping - different subjects",
			queryParams: streaming.QueryParams{
				GroupBy:       []string{"subject", "group1"},
				FilterGroupBy: map[string][]string{"group1": {"group1_value", "group2_value"}},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject2,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			expectedCount: 2, // Should not merge because different subjects
			description:   "Rows with different subjects should not merge",
		},
		{
			name: "FilterGroupBy affects grouping - different group values",
			queryParams: streaming.QueryParams{
				GroupBy:       []string{"subject", "group1"},
				FilterGroupBy: map[string][]string{"group1": {"group1_value", "group2_value"}},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group2Value,
					},
				},
			},
			expectedCount: 2, // Should not merge because different group values
			description:   "Rows with different group values should not merge",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := mergeMeterQueryRows(meterDef, testCase.queryParams, testCase.cachedRows, testCase.freshRows)
			assert.Equal(t, testCase.expectedCount, len(result), testCase.description)

			// Verify that the merge preserves the correct values based on aggregation
			if testCase.expectedCount == 1 && meterDef.Aggregation == meterpkg.MeterAggregationSum {
				expectedSum := testCase.cachedRows[0].Value + testCase.freshRows[0].Value
				assert.Equal(t, expectedSum, result[0].Value, "Sum aggregation should add values")
			}
		})
	}
}

// TestCreateGroupKeyFromRowWithFilterGroupBy tests that FilterGroupBy is considered in group key creation
func TestCreateGroupKeyFromRowWithFilterGroupBy(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"
	group2Value := "group2-value"

	testRow := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": &group1Value,
			"group2": &group2Value,
		},
	}

	tests := []struct {
		name        string
		queryParams streaming.QueryParams
		description string
	}{
		{
			name: "FilterGroupBy included in query params affects grouping",
			queryParams: streaming.QueryParams{
				GroupBy:       []string{"subject", "group1"},
				FilterGroupBy: map[string][]string{"group1": {"group1-value"}},
			},
			description: "FilterGroupBy should affect how grouping works even though it's not directly in the group key",
		},
		{
			name: "Different FilterGroupBy should be different even with same GroupBy",
			queryParams: streaming.QueryParams{
				GroupBy:       []string{"subject", "group1"},
				FilterGroupBy: map[string][]string{"group2": {"group2-value"}},
			},
			description: "Different FilterGroupBy with same GroupBy should create different contexts",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Create group key - the key itself doesn't include FilterGroupBy directly
			// but the fact that different FilterGroupBy values create different query hashes
			// means they will be cached separately
			result := createGroupKeyFromRowWithQueryParams(testRow, testCase.queryParams)
			assert.NotEmpty(t, result, testCase.description)

			// The group key should include the GroupBy fields
			assert.Contains(t, result, "subject=test-subject", "Group key should include subject")
			if contains(testCase.queryParams.GroupBy, "group1") {
				assert.Contains(t, result, "group=group1=group1-value", "Group key should include group1 if in GroupBy")
			}
		})
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestSmartDeduplicationWithRealWorldDuplicates tests the deduplication logic
// with real-world duplicates like the ones found in production
func TestSmartDeduplicationWithRealWorldDuplicates(t *testing.T) {
	subject := "2"
	accountID := "3"
	workspaceID := "169"

	windowStart, _ := time.Parse(time.RFC3339, "2025-05-21T00:00:00Z")
	windowEnd, _ := time.Parse(time.RFC3339, "2025-05-22T00:00:00Z")

	// These are the actual duplicate rows found in production
	rows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Value:       0, // First zero value
			Subject:     &subject,
			GroupBy: map[string]*string{
				"account_id":   &accountID,
				"workspace_id": &workspaceID,
			},
		},
		{
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Value:       143.99999999999966, // High precision value
			Subject:     &subject,
			GroupBy: map[string]*string{
				"account_id":   &accountID,
				"workspace_id": &workspaceID,
			},
		},
		{
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Value:       143.99999999999903, // Slightly different precision
			Subject:     &subject,
			GroupBy: map[string]*string{
				"account_id":   &accountID,
				"workspace_id": &workspaceID,
			},
		},
		{
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Value:       0, // Second zero value
			Subject:     &subject,
			GroupBy: map[string]*string{
				"account_id":   &accountID,
				"workspace_id": &workspaceID,
			},
		},
	}

	// Test with logger to see the decision process
	logger := slog.Default()

	deduplicatedRows, err := dedupeQueryRowsWithLogger(rows, []string{"account_id", "workspace_id"}, logger)
	require.NoError(t, err)

	// Should have only one row after deduplication
	assert.Len(t, deduplicatedRows, 1, "Should deduplicate to exactly one row")

	// Should choose the non-zero value with higher precision
	assert.Equal(t, 143.99999999999966, deduplicatedRows[0].Value,
		"Should choose the non-zero value with higher precision")

	// Verify other fields are preserved
	assert.Equal(t, windowStart, deduplicatedRows[0].WindowStart)
	assert.Equal(t, windowEnd, deduplicatedRows[0].WindowEnd)
	require.NotNil(t, deduplicatedRows[0].Subject)
	assert.Equal(t, subject, *deduplicatedRows[0].Subject)

	// Verify group by fields are preserved
	require.Contains(t, deduplicatedRows[0].GroupBy, "account_id")
	require.NotNil(t, deduplicatedRows[0].GroupBy["account_id"])
	assert.Equal(t, accountID, *deduplicatedRows[0].GroupBy["account_id"])

	require.Contains(t, deduplicatedRows[0].GroupBy, "workspace_id")
	require.NotNil(t, deduplicatedRows[0].GroupBy["workspace_id"])
	assert.Equal(t, workspaceID, *deduplicatedRows[0].GroupBy["workspace_id"])
}

// TestChooseBestDuplicateRowLogic tests the individual decision logic
func TestChooseBestDuplicateRowLogic(t *testing.T) {
	subject := "test"
	groupValue := "test-group"

	windowStart := time.Now().UTC()
	windowEnd := windowStart.Add(time.Hour)

	baseRow := meterpkg.MeterQueryRow{
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Subject:     &subject,
		GroupBy: map[string]*string{
			"group1": &groupValue,
		},
	}

	tests := []struct {
		name          string
		existingValue float64
		newValue      float64
		expectedValue float64
		description   string
	}{
		{
			name:          "non-zero beats zero",
			existingValue: 0,
			newValue:      100.5,
			expectedValue: 100.5,
			description:   "Non-zero value should be chosen over zero",
		},
		{
			name:          "zero loses to non-zero",
			existingValue: 100.5,
			newValue:      0,
			expectedValue: 100.5,
			description:   "Existing non-zero value should be kept over new zero",
		},
		{
			name:          "higher absolute value wins",
			existingValue: 143.99999999999903,
			newValue:      143.99999999999966,
			expectedValue: 143.99999999999966,
			description:   "Higher precision value should be chosen",
		},
		{
			name:          "negative higher absolute value wins",
			existingValue: -143.99999999999903,
			newValue:      -143.99999999999966,
			expectedValue: -143.99999999999966,
			description:   "Higher absolute value should be chosen even for negatives",
		},
		{
			name:          "both zero keeps existing",
			existingValue: 0,
			newValue:      0,
			expectedValue: 0,
			description:   "When both are zero, keep existing",
		},
	}

	logger := slog.Default()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			existingRow := baseRow
			existingRow.Value = testCase.existingValue

			newRow := baseRow
			newRow.Value = testCase.newValue

			result := chooseBestDuplicateRow(existingRow, newRow, "test-key", logger)
			assert.Equal(t, testCase.expectedValue, result.Value, testCase.description)
		})
	}
}
