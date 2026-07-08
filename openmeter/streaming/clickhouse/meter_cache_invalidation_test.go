package clickhouse

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestLateEventWindows(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	age := time.Hour

	newEvent := func(namespace, eventType string, at time.Time) streaming.RawEvent {
		return streaming.RawEvent{Namespace: namespace, Type: eventType, Time: at}
	}

	t.Run("NoLateEvents", func(t *testing.T) {
		windows, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", now.Add(-10*time.Minute)),
			newEvent("ns1", "api-calls", now),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Empty(t, windows)
	})

	t.Run("EventExactlyAtCutoffIsNotLate", func(t *testing.T) {
		// The cutoff is exclusive: an event exactly minimumUsageAge old sits at the
		// freshness horizon the reader always serves live, so no marker is needed.
		windows, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", now.Add(-age)),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Empty(t, windows)

		windows, err = lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", now.Add(-age).Add(-time.Second)),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Len(t, windows, 1)
	})

	t.Run("MergesMinAndMaxBucketsPerNamespaceAndType", func(t *testing.T) {
		windows, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", now.Add(-3*time.Hour).Add(10*time.Minute)),
			newEvent("ns1", "api-calls", now.Add(-6*time.Hour).Add(59*time.Minute)),
			newEvent("ns1", "api-calls", now.Add(-4*time.Hour)),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Equal(t, []invalidationWindow{
			{
				Namespace: "ns1",
				EventType: "api-calls",
				WindowLo:  now.Add(-6 * time.Hour),
				WindowHi:  now.Add(-2 * time.Hour),
			},
		}, windows)
	})

	t.Run("SeparatesNamespacesAndTypesSorted", func(t *testing.T) {
		windows, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns2", "api-calls", now.Add(-3*time.Hour)),
			newEvent("ns1", "tokens", now.Add(-3*time.Hour)),
			newEvent("ns1", "api-calls", now.Add(-3*time.Hour)),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Equal(t, []invalidationWindow{
			{Namespace: "ns1", EventType: "api-calls", WindowLo: now.Add(-3 * time.Hour), WindowHi: now.Add(-2 * time.Hour)},
			{Namespace: "ns1", EventType: "tokens", WindowLo: now.Add(-3 * time.Hour), WindowHi: now.Add(-2 * time.Hour)},
			{Namespace: "ns2", EventType: "api-calls", WindowLo: now.Add(-3 * time.Hour), WindowHi: now.Add(-2 * time.Hour)},
		}, windows)
	})

	t.Run("DayGrainTruncatesToUTCDays", func(t *testing.T) {
		windows, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", time.Date(2026, 7, 1, 18, 30, 0, 0, time.UTC)),
		}, now, age, CacheGrainDay)
		require.NoError(t, err)
		require.Equal(t, []invalidationWindow{
			{
				Namespace: "ns1",
				EventType: "api-calls",
				WindowLo:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				WindowHi:  time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
			},
		}, windows)
	})

	t.Run("NonUTCEventTimesAlignToUTCBuckets", func(t *testing.T) {
		// Cache buckets are UTC-aligned; an event carrying a zoned wall clock must land in
		// the bucket of its UTC instant.
		budapest, err := time.LoadLocation("Europe/Budapest")
		require.NoError(t, err)

		windows, err := lateEventWindows([]streaming.RawEvent{
			// 08:30 CEST == 06:30 UTC
			newEvent("ns1", "api-calls", time.Date(2026, 7, 3, 8, 30, 0, 0, budapest)),
		}, now, age, CacheGrainHour)
		require.NoError(t, err)
		require.Equal(t, []invalidationWindow{
			{
				Namespace: "ns1",
				EventType: "api-calls",
				WindowLo:  time.Date(2026, 7, 3, 6, 0, 0, 0, time.UTC),
				WindowHi:  time.Date(2026, 7, 3, 7, 0, 0, 0, time.UTC),
			},
		}, windows)
	})

	t.Run("InvalidGrain", func(t *testing.T) {
		_, err := lateEventWindows([]streaming.RawEvent{
			newEvent("ns1", "api-calls", now.Add(-3*time.Hour)),
		}, now, age, CacheGrain("fortnight"))
		require.Error(t, err)
	})
}

// TestInsertInvalidationMarkersToSQL is the structural proof of G6: created_at must not
// appear in the INSERT column list, so ClickHouse stamps it from the table's DEFAULT
// now64(3) (server time) instead of an app clock.
func TestInsertInvalidationMarkersToSQL(t *testing.T) {
	windowLo := time.Date(2026, 7, 3, 6, 0, 0, 0, time.UTC)
	windowHi := windowLo.Add(2 * time.Hour)

	sql, args := insertInvalidationMarkers{
		Database: "openmeter",
		Windows: []invalidationWindow{
			{Namespace: "ns1", EventType: "api-calls", WindowLo: windowLo, WindowHi: windowHi},
			{Namespace: "ns2", EventType: "tokens", WindowLo: windowLo, WindowHi: windowHi},
		},
	}.toSQL()

	require.Equal(t, "INSERT INTO openmeter.om_meter_cache_invalidations (namespace, event_type, window_lo, window_hi) VALUES (?, ?, ?, ?), (?, ?, ?, ?)", sql)
	require.NotContains(t, sql, "created_at")
	require.Equal(t, []interface{}{
		"ns1", "api-calls", windowLo, windowHi,
		"ns2", "tokens", windowLo, windowHi,
	}, args)
}

func TestRefreshThrottler(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	t.Run("PerViewMinInterval", func(t *testing.T) {
		throttler := newRefreshThrottler(10 * time.Minute)

		require.True(t, throttler.allow("view_a", now))
		require.False(t, throttler.allow("view_a", now))
		require.False(t, throttler.allow("view_a", now.Add(10*time.Minute-time.Second)))
		require.True(t, throttler.allow("view_a", now.Add(10*time.Minute)))

		// Distinct views throttle independently
		require.True(t, throttler.allow("view_b", now))
	})

	t.Run("ConcurrentCallersGetOneSlot", func(t *testing.T) {
		throttler := newRefreshThrottler(10 * time.Minute)

		var wg sync.WaitGroup
		var allowed atomic.Uint64

		for range 50 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				if throttler.allow("view_a", now) {
					allowed.Add(1)
				}
			}()
		}

		wg.Wait()
		require.Equal(t, uint64(1), allowed.Load())
	})
}

func TestAffectedViewNames(t *testing.T) {
	metadataFor := func(namespace, eventType string) meterCacheMVMetadata {
		return meterCacheMVMetadata{
			Namespace: namespace,
			MeterKey:  "meter-1",
			EventType: eventType,
			MeterHash: formatCacheHash(1),
			DDLHash:   formatCacheHash(2),
		}
	}

	views := []deployedCacheMV{
		{Name: mvName("ns1", 1), Metadata: metadataFor("ns1", "api-calls")},
		{Name: mvName("ns1", 2), Metadata: metadataFor("ns1", "api-calls")},
		{Name: mvName("ns1", 3), Metadata: metadataFor("ns1", "tokens")},
		{Name: mvName("ns2", 4), Metadata: metadataFor("ns2", "api-calls")},
	}

	t.Run("MatchesExactNamespaceAndEventType", func(t *testing.T) {
		names := affectedViewNames(views, []invalidationWindow{
			{Namespace: "ns1", EventType: "api-calls"},
		})

		expected := []string{mvName("ns1", 1), mvName("ns1", 2)}
		slices.Sort(expected)
		require.Equal(t, expected, names)
	})

	t.Run("NoMatchForUnknownPair", func(t *testing.T) {
		require.Empty(t, affectedViewNames(views, []invalidationWindow{
			{Namespace: "ns3", EventType: "api-calls"},
			{Namespace: "ns2", EventType: "tokens"},
		}))
	})

	t.Run("DeduplicatesAcrossWindows", func(t *testing.T) {
		names := affectedViewNames(views, []invalidationWindow{
			{Namespace: "ns1", EventType: "tokens"},
			{Namespace: "ns1", EventType: "tokens"},
		})
		require.Equal(t, []string{mvName("ns1", 3)}, names)
	})
}

func withCacheEnabled(c Config) Config {
	c.Cache = CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	return c
}

// TestBatchInsertMarkerFailureDoesNotFailIngest forces both the invalidation marker insert
// and the view listing to fail and proves BatchInsert still succeeds: invalidation is
// best-effort, failing ingestion over it would make callers re-send already-stored events.
func TestBatchInsertMarkerFailureDoesNotFailIngest(t *testing.T) {
	connector, mockCH := GetMockConnector(t, withCacheEnabled)

	// The events insert itself succeeds
	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.events")
	}), mock.Anything).Return(nil).Once()

	// The invalidation marker insert fails
	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.om_meter_cache_invalidations")
	}), mock.Anything).Return(errors.New("marker insert failed")).Once()

	// The system.tables listing for refresh triggering fails too
	mockCH.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.Contains(sql, "system.tables")
	}), mock.Anything).Return(NewMockRows(), errors.New("listing failed")).Once()

	err := connector.BatchInsert(t.Context(), []streaming.RawEvent{{
		Namespace: "ns1",
		Type:      "api-calls",
		Subject:   "subject-1",
		Time:      time.Now().UTC().Add(-3 * time.Hour),
		Data:      `{"value": 1}`,
	}})
	require.NoError(t, err)

	require.Equal(t, uint64(1), connector.cacheInvalidator.markerInsertFailures.Load())
	require.Equal(t, uint64(1), connector.cacheInvalidator.refreshTriggerFailures.Load())
	require.Equal(t, uint64(0), connector.cacheInvalidator.refreshTriggersFired.Load())
	mockCH.AssertExpectations(t)
}

// TestBatchInsertLateEventTriggersThrottledRefresh drives two late batches through
// BatchInsert and asserts: markers are written per batch, the system.tables listing is
// scanned once (TTL cache, G7), and SYSTEM REFRESH VIEW fires exactly once (throttle).
func TestBatchInsertLateEventTriggersThrottledRefresh(t *testing.T) {
	connector, mockCH := GetMockConnector(t, withCacheEnabled)

	viewName := mvName("ns1", 1)
	comment, err := meterCacheMVMetadata{
		Namespace: "ns1",
		MeterKey:  "meter-1",
		EventType: "api-calls",
		MeterHash: formatCacheHash(1),
		DDLHash:   formatCacheHash(2),
	}.marshal()
	require.NoError(t, err)

	// Events and marker inserts succeed for both batches
	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.events")
	}), mock.Anything).Return(nil).Twice()
	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.om_meter_cache_invalidations")
	}), mock.Anything).Return(nil).Twice()

	// The system.tables listing is served exactly once; the second batch arrives within
	// the listing TTL and must reuse the in-process snapshot
	listRows := NewMockRows()
	listRows.On("Next").Return(true).Once()
	listRows.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(0).([]interface{})
		*(dest[0].(*string)) = viewName
		*(dest[1].(*string)) = comment
	}).Return(nil).Once()
	listRows.On("Next").Return(false).Once()
	listRows.On("Err").Return(nil).Once()
	listRows.On("Close").Return(nil).Once()

	mockCH.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.Contains(sql, "system.tables")
	}), mock.Anything).Return(listRows, nil).Once()

	// The refresh trigger fires exactly once across both batches (throttle window is the
	// refresh interval)
	mockCH.On("Exec", mock.Anything, "SYSTEM REFRESH VIEW testdb."+viewName, mock.Anything).Return(nil).Once()

	lateEvent := streaming.RawEvent{
		Namespace: "ns1",
		Type:      "api-calls",
		Subject:   "subject-1",
		Time:      time.Now().UTC().Add(-3 * time.Hour),
		Data:      `{"value": 1}`,
	}

	require.NoError(t, connector.BatchInsert(t.Context(), []streaming.RawEvent{lateEvent}))
	require.NoError(t, connector.BatchInsert(t.Context(), []streaming.RawEvent{lateEvent}))

	require.Equal(t, uint64(1), connector.cacheInvalidator.refreshTriggersFired.Load())
	require.Equal(t, uint64(0), connector.cacheInvalidator.markerInsertFailures.Load())
	require.Equal(t, uint64(0), connector.cacheInvalidator.refreshTriggerFailures.Load())
	mockCH.AssertExpectations(t)
}

// TestBatchInsertOnTimeEventsSkipInvalidation proves a batch without late events performs
// only the events insert: no marker, no listing, no refresh trigger.
func TestBatchInsertOnTimeEventsSkipInvalidation(t *testing.T) {
	connector, mockCH := GetMockConnector(t, withCacheEnabled)

	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.events")
	}), mock.Anything).Return(nil).Once()

	require.NoError(t, connector.BatchInsert(t.Context(), []streaming.RawEvent{{
		Namespace: "ns1",
		Type:      "api-calls",
		Subject:   "subject-1",
		Time:      time.Now().UTC().Add(-10 * time.Minute),
		Data:      `{"value": 1}`,
	}}))

	mockCH.AssertExpectations(t)
	mockCH.AssertNumberOfCalls(t, "Exec", 1)
	mockCH.AssertNumberOfCalls(t, "Query", 0)
}

// TestBatchInsertCacheDisabledSkipsInvalidation proves the hook is inert when the cache is
// disabled: no invalidator is constructed and late events cause no extra statements.
func TestBatchInsertCacheDisabledSkipsInvalidation(t *testing.T) {
	connector, mockCH := GetMockConnector(t)
	require.Nil(t, connector.cacheInvalidator)

	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.HasPrefix(sql, "INSERT INTO testdb.events")
	}), mock.Anything).Return(nil).Once()

	require.NoError(t, connector.BatchInsert(t.Context(), []streaming.RawEvent{{
		Namespace: "ns1",
		Type:      "api-calls",
		Subject:   "subject-1",
		Time:      time.Now().UTC().Add(-3 * time.Hour),
		Data:      `{"value": 1}`,
	}}))

	mockCH.AssertExpectations(t)
	mockCH.AssertNumberOfCalls(t, "Exec", 1)
}
