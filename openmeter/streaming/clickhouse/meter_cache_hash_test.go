package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func pinnedHashMeter() meter.Meter {
	return meter.Meter{
		EventType:     "api-calls",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.tokens"),
		GroupBy:       map[string]string{"model": "$.model", "region": "$.region"},
	}
}

func TestMeterHash(t *testing.T) {
	base := pinnedHashMeter()

	t.Run("pinned value", func(t *testing.T) {
		// Cached rows are keyed by this hash in production: an unintended algorithm or
		// input-ordering change would orphan every cached row (and re-backfill every
		// meter), so the exact value is pinned. Update the constant only for a
		// deliberate, migration-aware hash change.
		require.Equal(t, uint64(0xfe20ef28411f6935), meterHash(base, CacheGrainHour))
	})

	t.Run("independent of group by map iteration order", func(t *testing.T) {
		// Maps iterate in random order; repeated calls only agree if keys are sorted.
		for range 100 {
			require.Equal(t, uint64(0xfe20ef28411f6935), meterHash(pinnedHashMeter(), CacheGrainHour))
		}
	})

	t.Run("grain is part of the shape (G4)", func(t *testing.T) {
		// A grain config change must read as a shape change so old-grain rows are
		// filtered out by hash instead of being co-read with new-grain rows.
		hour := meterHash(base, CacheGrainHour)
		require.NotEqual(t, hour, meterHash(base, CacheGrainMinute))
		require.NotEqual(t, hour, meterHash(base, CacheGrainDay))
	})

	t.Run("sensitive to every shape input", func(t *testing.T) {
		pinned := meterHash(base, CacheGrainHour)

		mutations := map[string]func(m *meter.Meter){
			"event type":           func(m *meter.Meter) { m.EventType = "other" },
			"aggregation":          func(m *meter.Meter) { m.Aggregation = meter.MeterAggregationAvg },
			"value property":       func(m *meter.Meter) { m.ValueProperty = lo.ToPtr("$.other") },
			"nil value property":   func(m *meter.Meter) { m.ValueProperty = nil },
			"group by path":        func(m *meter.Meter) { m.GroupBy["model"] = "$.other" },
			"group by key added":   func(m *meter.Meter) { m.GroupBy["extra"] = "$.extra" },
			"group by key removed": func(m *meter.Meter) { delete(m.GroupBy, "region") },
			"empty group by":       func(m *meter.Meter) { m.GroupBy = nil },
		}

		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				m := pinnedHashMeter()
				mutate(&m)
				require.NotEqual(t, pinned, meterHash(m, CacheGrainHour))
			})
		}
	})

	t.Run("insensitive to non-shape fields", func(t *testing.T) {
		m := pinnedHashMeter()
		m.Key = "renamed-meter"
		m.EventFrom = lo.ToPtr(time.Now())
		require.Equal(t, meterHash(base, CacheGrainHour), meterHash(m, CacheGrainHour))
	})

	t.Run("component boundaries do not bleed", func(t *testing.T) {
		a := meter.Meter{EventType: "e", Aggregation: meter.MeterAggregationCount, GroupBy: map[string]string{"ab": "cd"}}
		b := meter.Meter{EventType: "e", Aggregation: meter.MeterAggregationCount, GroupBy: map[string]string{"a": "bcd"}}
		require.NotEqual(t, meterHash(a, CacheGrainHour), meterHash(b, CacheGrainHour))

		// nil vs empty value property differ via the presence marker
		c := meter.Meter{EventType: "e", Aggregation: meter.MeterAggregationCount, ValueProperty: lo.ToPtr("")}
		d := meter.Meter{EventType: "e", Aggregation: meter.MeterAggregationCount}
		require.NotEqual(t, meterHash(c, CacheGrainHour), meterHash(d, CacheGrainHour))
	})
}

func TestMVName(t *testing.T) {
	hash := meterHash(pinnedHashMeter(), CacheGrainHour)

	t.Run("pinned format", func(t *testing.T) {
		// The reconciler discovers cache MVs by this prefix and correlates them to
		// meters by the hash16 suffix; the format is part of the deployed contract.
		require.Equal(t, "om_meter_cache_mv_933b5bde_fe20ef28411f6935", mvName("default", hash))
	})

	t.Run("namespace changes the name", func(t *testing.T) {
		require.NotEqual(t, mvName("default", hash), mvName("other", hash))
	})

	t.Run("always carries the reconciler scan prefix", func(t *testing.T) {
		assert.Regexp(t, `^om_meter_cache_mv_[0-9a-f]{8}_[0-9a-f]{16}$`, mvName("Any Namespace ✔", hash))
	})
}

func TestDDLHash(t *testing.T) {
	eventFrom, err := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	require.NoError(t, err)

	base := ddlHash(CacheGrainHour, 10*time.Minute, time.Hour, nil, "SELECT 1")

	t.Run("pinned value", func(t *testing.T) {
		// Pinned for the same reason as meterHash: an accidental change would make the
		// reconciler drop and recreate (and possibly re-backfill) every deployed MV at
		// once on the next pass.
		require.Equal(t, uint64(0x863516d3145c3294), base)
	})

	t.Run("sensitive to every input", func(t *testing.T) {
		require.NotEqual(t, base, ddlHash(CacheGrainMinute, 10*time.Minute, time.Hour, nil, "SELECT 1"))
		require.NotEqual(t, base, ddlHash(CacheGrainHour, 5*time.Minute, time.Hour, nil, "SELECT 1"))
		require.NotEqual(t, base, ddlHash(CacheGrainHour, 10*time.Minute, 2*time.Hour, nil, "SELECT 1"))
		require.NotEqual(t, base, ddlHash(CacheGrainHour, 10*time.Minute, time.Hour, &eventFrom, "SELECT 1"))
		require.NotEqual(t, base, ddlHash(CacheGrainHour, 10*time.Minute, time.Hour, nil, "SELECT 2"))
	})
}

func TestCacheHashFormat(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		for _, hash := range []uint64{0, 1, 0xfe20ef28411f6935, ^uint64(0)} {
			formatted := formatCacheHash(hash)
			require.Len(t, formatted, 16)

			parsed, err := parseCacheHash(formatted)
			require.NoError(t, err)
			require.Equal(t, hash, parsed)
		}
	})

	t.Run("rejects non-canonical widths and non-hex", func(t *testing.T) {
		_, err := parseCacheHash("2a")
		require.ErrorContains(t, err, "16 hex characters")

		_, err = parseCacheHash("zzzzzzzzzzzzzzzz")
		require.ErrorContains(t, err, "not valid hex")
	})
}

func TestMeterCacheMVMetadata(t *testing.T) {
	backfilledAt, err := time.Parse(time.RFC3339, "2026-07-03T09:00:00Z")
	require.NoError(t, err)

	valid := meterCacheMVMetadata{
		Namespace: "ns1",
		MeterKey:  "meter1",
		EventType: "event1",
		MeterHash: formatCacheHash(0xfe20ef28411f6935),
		DDLHash:   formatCacheHash(0x863516d3145c3294),
	}

	t.Run("marshal is stable and unstamped by default", func(t *testing.T) {
		comment, err := valid.marshal()
		require.NoError(t, err)
		// backfilled_at must be absent (not null) while unstamped: its presence is the
		// reader's G3 gate. covered_at likewise starts absent and only appears once the
		// reconciler advances the coverage watermark.
		require.Equal(t, `{"namespace":"ns1","meter_key":"meter1","event_type":"event1","meter_hash":"fe20ef28411f6935","ddl_hash":"863516d3145c3294"}`, comment)
	})

	t.Run("round trip with backfilled_at and covered_at stamps", func(t *testing.T) {
		stamped := valid
		stamped.BackfilledAt = &backfilledAt
		stamped.CoveredAt = lo.ToPtr(backfilledAt.Add(30 * time.Minute))

		comment, err := stamped.marshal()
		require.NoError(t, err)

		parsed, err := parseMeterCacheMVMetadata(comment)
		require.NoError(t, err)
		require.Equal(t, stamped, parsed)
	})

	t.Run("round trip without stamp", func(t *testing.T) {
		comment, err := valid.marshal()
		require.NoError(t, err)

		parsed, err := parseMeterCacheMVMetadata(comment)
		require.NoError(t, err)
		require.Equal(t, valid, parsed)
		require.Nil(t, parsed.BackfilledAt)
	})

	t.Run("parse rejects foreign comments", func(t *testing.T) {
		tests := []struct {
			name    string
			comment string
			wantErr string
		}{
			{name: "not json", comment: "some human comment", wantErr: "unmarshal"},
			{name: "empty json", comment: "{}", wantErr: "meter_key is required"},
			// A namespace-less comment is a pre-namespace-metadata deployment: the reader
			// cannot tell whether it belongs to the querying namespace or a name-fold
			// colliding one, so it must be treated as foreign and recreated.
			{name: "missing namespace", comment: `{"meter_key":"m","event_type":"e","meter_hash":"fe20ef28411f6935","ddl_hash":"863516d3145c3294"}`, wantErr: "namespace is required"},
			{name: "missing event type", comment: `{"namespace":"ns1","meter_key":"m","meter_hash":"fe20ef28411f6935","ddl_hash":"863516d3145c3294"}`, wantErr: "event_type is required"},
			{name: "short hash", comment: `{"namespace":"ns1","meter_key":"m","event_type":"e","meter_hash":"2a","ddl_hash":"863516d3145c3294"}`, wantErr: "meter_hash"},
			{name: "non-hex ddl hash", comment: `{"namespace":"ns1","meter_key":"m","event_type":"e","meter_hash":"fe20ef28411f6935","ddl_hash":"zzzzzzzzzzzzzzzz"}`, wantErr: "ddl_hash"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := parseMeterCacheMVMetadata(tt.comment)
				require.ErrorContains(t, err, tt.wantErr)
			})
		}
	})

	t.Run("marshal refuses invalid metadata", func(t *testing.T) {
		invalid := valid
		invalid.MeterHash = "2a"

		_, err := invalid.marshal()
		require.ErrorContains(t, err, "meter_hash")
	})
}
