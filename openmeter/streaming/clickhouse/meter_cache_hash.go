package clickhouse

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"maps"
	"slices"
	"strconv"
	"time"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// meterHash identifies the cached shape of a meter: which rows in om_meter_cache belong
// to this meter's current definition. It covers everything that changes what a cached row
// means — event type, aggregation, value property, the group-by dimension map, and the
// cache grain. The grain is included deliberately (a grain config change reads as a shape
// change): rows bucketed at the old grain must be filtered out by hash and GC'd, never
// co-read with rows at the new grain.
//
// The meter key and namespace are intentionally excluded: rows carry both as columns and
// every read co-filters on namespace, so the hash only needs to capture shape.
func meterHash(m meterpkg.Meter, grain CacheGrain) uint64 {
	h := fnv.New64a()

	hashComponent(h, m.EventType)
	hashComponent(h, string(m.Aggregation))

	// Presence marker distinguishes a nil value property from an empty one and anchors
	// the section so group-by content cannot bleed into it.
	if m.ValueProperty == nil {
		hashComponent(h, "0")
	} else {
		hashComponent(h, "1")
		hashComponent(h, *m.ValueProperty)
	}

	keys := slices.Sorted(maps.Keys(m.GroupBy))

	// The pair count anchors the group-by section so its content cannot bleed into the
	// grain component.
	hashComponent(h, strconv.Itoa(len(keys)))

	for _, key := range keys {
		hashComponent(h, key)
		hashComponent(h, m.GroupBy[key])
	}

	hashComponent(h, string(grain))

	return h.Sum64()
}

// hashComponent writes s length-prefixed so component boundaries stay unambiguous:
// ("ab", "c") must never hash equal to ("a", "bc").
func hashComponent(h hash.Hash64, s string) {
	// hash.Hash Write never returns an error
	_, _ = fmt.Fprintf(h, "%d:%s", len(s), s)
}

// mvNamePrefixForNamespace returns the name prefix shared by every cache MV of one
// namespace: om_meter_cache_mv_<ns8>_. The namespace is folded to 8 hex chars because
// namespace strings can be long and contain characters invalid in identifiers. Distinct
// namespaces can collide on the folded prefix, which is acceptable because the prefix is
// only used for discovery (refresh triggering, reconciler scans) where a collision at
// worst touches another namespace's views; data reads are keyed by the full namespace and
// meter_hash columns, never by this name.
func mvNamePrefixForNamespace(namespace string) string {
	ns := fnv.New32a()
	// hash.Hash Write never returns an error
	_, _ = ns.Write([]byte(namespace))

	return fmt.Sprintf("%s%08x_", meterCacheMVNamePrefix, ns.Sum32())
}

// mvName returns the deterministic name of the cache MV for a meter shape in a namespace:
// om_meter_cache_mv_<ns8>_<hash16>, where hash is the meterHash of the shape. Data safety
// does not depend on this name (rows are keyed by namespace + meter_hash), it only has to
// be unique enough for the reconciler's system.tables prefix scan.
func mvName(namespace string, hash uint64) string {
	return fmt.Sprintf("%s%016x", mvNamePrefixForNamespace(namespace), hash)
}

// ddlHash detects drift between a deployed MV and the DDL the generator would emit today,
// beyond what meterHash covers: refresh cadence, freshness horizon, EventFrom, and the
// full generated SELECT (so any generator change is treated as drift). The reconciler
// drops and recreates the MV when it differs; whether that recreate also needs a
// re-backfill depends on which input moved, which is why the inputs are hashed alongside
// the SELECT instead of relying on the SELECT text alone.
func ddlHash(grain CacheGrain, refreshInterval, minimumUsageAge time.Duration, eventFrom *time.Time, selectSQL string) uint64 {
	h := fnv.New64a()

	hashComponent(h, string(grain))
	hashComponent(h, refreshInterval.String())
	hashComponent(h, minimumUsageAge.String())

	if eventFrom == nil {
		hashComponent(h, "0")
	} else {
		hashComponent(h, "1")
		hashComponent(h, strconv.FormatInt(eventFrom.Unix(), 10))
	}

	hashComponent(h, selectSQL)

	return h.Sum64()
}

// formatCacheHash renders a meterHash or ddlHash for the MV comment metadata. Hashes are
// stored as fixed-width hex strings, not JSON numbers, because uint64 values above 2^53
// do not survive a JSON number round-trip through arbitrary tooling.
func formatCacheHash(hash uint64) string {
	return fmt.Sprintf("%016x", hash)
}

// parseCacheHash parses a hash formatted by formatCacheHash. The fixed 16-char width is
// enforced so hand-edited or foreign comments fail parsing instead of silently comparing
// unequal to freshly formatted hashes.
func parseCacheHash(s string) (uint64, error) {
	if len(s) != 16 {
		return 0, fmt.Errorf("cache hash must be 16 hex characters, got %d", len(s))
	}

	hash, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("cache hash is not valid hex: %w", err)
	}

	return hash, nil
}

// meterCacheMVMetadata is the JSON document stored in the cache MV's COMMENT. It is the
// reconciler's and reader's only source of truth about a deployed MV: which meter it
// serves, which shape (meter_hash) and generator output (ddl_hash) it was created from,
// and whether its backfill completed.
//
// BackfilledAt is nil until the one-time backfill finishes; it is stamped afterwards via
// ALTER TABLE ... MODIFY COMMENT. Readers must refuse the cache while it is unstamped: an
// MV that exists but was never backfilled only contains recently refreshed buckets, and
// serving it would silently drop all older history from query results.
type meterCacheMVMetadata struct {
	MeterKey     string     `json:"meter_key"`
	EventType    string     `json:"event_type"`
	MeterHash    string     `json:"meter_hash"`
	DDLHash      string     `json:"ddl_hash"`
	BackfilledAt *time.Time `json:"backfilled_at,omitempty"`
}

func (m meterCacheMVMetadata) Validate() error {
	var errs []error

	if m.MeterKey == "" {
		errs = append(errs, errors.New("meter_key is required"))
	}

	if m.EventType == "" {
		errs = append(errs, errors.New("event_type is required"))
	}

	if _, err := parseCacheHash(m.MeterHash); err != nil {
		errs = append(errs, fmt.Errorf("meter_hash: %w", err))
	}

	if _, err := parseCacheHash(m.DDLHash); err != nil {
		errs = append(errs, fmt.Errorf("ddl_hash: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (m meterCacheMVMetadata) marshal() (string, error) {
	if err := m.Validate(); err != nil {
		return "", fmt.Errorf("validate meter cache mv metadata: %w", err)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal meter cache mv metadata: %w", err)
	}

	return string(data), nil
}

// parseMeterCacheMVMetadata parses an MV comment read back from system.tables. A non-nil
// error means the view must be treated as foreign or corrupt (read live, let the
// reconciler repair it), never trusted as a healthy cache MV.
func parseMeterCacheMVMetadata(comment string) (meterCacheMVMetadata, error) {
	var metadata meterCacheMVMetadata

	if err := json.Unmarshal([]byte(comment), &metadata); err != nil {
		return meterCacheMVMetadata{}, fmt.Errorf("unmarshal meter cache mv metadata: %w", err)
	}

	if err := metadata.Validate(); err != nil {
		return meterCacheMVMetadata{}, fmt.Errorf("validate meter cache mv metadata: %w", err)
	}

	return metadata, nil
}
