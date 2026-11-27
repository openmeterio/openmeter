package migrate_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestNotificationRuleFeatureIDsMigration(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	db := testDB.PGDriver.DB()

	scripts := make(map[string]string)
	for _, direction := range []string{"up", "down"} {
		data, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, "20261125160000_notification_rule_feature_ids."+direction+".sql"))
		require.NoError(t, err)

		scripts[direction] = string(data)
	}

	preflight, err := os.ReadFile("notification_rule_feature_ids_preflight.sql")
	require.NoError(t, err)

	const (
		featureA = "01G65Z755AFWAKHE12NY0CQ9FA"
		featureB = "01G65Z755AFWAKHE12NY0CQ9FB"
		featureC = "01G65Z755AFWAKHE12NY0CQ9FC"
	)
	tests := []struct {
		name, config, expected, wantError   string
		archived, reuse, collision, foreign bool
		enabled                             bool
		dropAction                          string
		featureDeleted, overlap             bool
		ruleHour                            *int
		deleteHour                          *int
		wantCandidateIDs                    []string
	}{
		{
			name:     "keys and ids preserve order and deduplicate",
			config:   `{"features":["b","` + featureA + `","a"]}`,
			expected: `{"features":[{"id":"` + featureB + `"},{"id":"` + featureA + `"}]}`,
		},
		{
			name:       "deleted after archival without key reuse retains archived ID",
			config:     `{"features":["a"]}`,
			expected:   `{"features":[{"id":"` + featureA + `"}]}`,
			archived:   true,
			deleteHour: new(3),
		},
		{
			name:             "deleted after key reuse flags both historical candidates",
			config:           `{"features":["a"]}`,
			reuse:            true,
			deleteHour:       new(3),
			wantError:        "ambiguous rule feature",
			wantCandidateIDs: []string{featureA, featureC},
		},
		{
			name:       "key reused at deletion cannot retarget deleted rule",
			config:     `{"features":["a"]}`,
			expected:   `{"features":[{"id":"` + featureA + `"}]}`,
			reuse:      true,
			deleteHour: new(2),
		},
		{
			name:       "feature archived before rule creation is excluded",
			config:     `{"features":["a"]}`,
			expected:   `{"features":[{"id":"` + featureC + `"}]}`,
			reuse:      true,
			ruleHour:   new(3),
			deleteHour: new(4),
		},
		{
			name:       "explicit ID stays pinned on deleted rule after key reuse",
			config:     `{"features":["` + featureA + `"]}`,
			expected:   `{"features":[{"id":"` + featureA + `"}]}`,
			reuse:      true,
			deleteHour: new(3),
		},
		{
			name:      "key resolves to pinned ID despite another feature using that ID as a key",
			config:    `{"features":["a"]}`,
			expected:  `{"features":[{"id":"` + featureA + `"}]}`,
			collision: true,
		},
		{
			name:      "object ID stays pinned when another feature uses the ID as a key",
			config:    `{"features":[{"id":"` + featureA + `"}]}`,
			expected:  `{"features":[{"id":"` + featureA + `"}]}`,
			collision: true,
		},
		{
			name:     "mixed references preserve first occurrence",
			config:   `{"features":[{"id":"` + featureB + `"},"a","b"]}`,
			expected: `{"features":[{"id":"` + featureB + `"},{"id":"` + featureA + `"}]}`,
		},
		{
			name:      "null array entry is malformed",
			config:    `{"features":[null]}`,
			wantError: "invalid rule feature reference",
		},
		{
			name:     "already resolved",
			config:   `{"features":["` + featureA + `"]}`,
			expected: `{"features":[{"id":"` + featureA + `"}]}`,
		},
		{
			name:       "only missing reference is dropped",
			config:     `{"features":["b","missing","a","b"]}`,
			expected:   `{"features":[{"id":"` + featureB + `"},{"id":"` + featureA + `"}]}`,
			dropAction: "drop_missing_feature",
			enabled:    true,
		},
		{
			name:     "omitted",
			config:   `{}`,
			expected: `{}`,
		},
		{
			name:     "null",
			config:   `{"features":null}`,
			expected: `{}`,
		},
		{
			name:     "empty",
			config:   `{"features":[]}`,
			expected: `{"features":[]}`,
		},
		{
			name:       "missing",
			config:     `{"features":["missing"]}`,
			expected:   `{"features":[]}`,
			dropAction: "drop_missing_feature_filter_becomes_catch_all",
			enabled:    true,
		},
		{
			name:       "cross namespace",
			config:     `{"features":["` + featureA + `"]}`,
			foreign:    true,
			expected:   `{"features":[]}`,
			dropAction: "drop_missing_feature_filter_becomes_catch_all",
		},
		{
			name:     "id stays pinned after key reuse",
			config:   `{"features":["` + featureA + `"]}`,
			expected: `{"features":[{"id":"` + featureA + `"}]}`,
			reuse:    true,
			ruleHour: new(3),
		},
		{
			name:     "reused key before archive",
			config:   `{"features":["a"]}`,
			expected: `{"features":[{"id":"` + featureA + `"}]}`,
			reuse:    true,
		},
		{
			name:     "reused key at replacement boundary",
			config:   `{"features":["a"]}`,
			expected: `{"features":[{"id":"` + featureC + `"}]}`,
			reuse:    true,
			ruleHour: new(2),
		},
		{
			name:     "key at creation boundary",
			config:   `{"features":["a"]}`,
			expected: `{"features":[{"id":"` + featureA + `"}]}`,
			ruleHour: new(0),
		},
		{
			name:       "key before creation",
			config:     `{"features":["a"]}`,
			ruleHour:   new(-1),
			expected:   `{"features":[]}`,
			dropAction: "drop_missing_feature_filter_becomes_catch_all",
		},
		{
			name:       "key at archive boundary",
			config:     `{"features":["a"]}`,
			archived:   true,
			ruleHour:   new(2),
			expected:   `{"features":[]}`,
			dropAction: "drop_missing_feature_filter_becomes_catch_all",
		},
		{
			name:           "key before deletion",
			config:         `{"features":["a"]}`,
			expected:       `{"features":[{"id":"` + featureA + `"}]}`,
			featureDeleted: true,
		},
		{
			name:           "key at deletion boundary",
			config:         `{"features":["a"]}`,
			featureDeleted: true,
			ruleHour:       new(2),
			expected:       `{"features":[]}`,
			dropAction:     "drop_missing_feature_filter_becomes_catch_all",
		},
		{
			name:      "overlapping feature lifetimes",
			config:    `{"features":["a"]}`,
			reuse:     true,
			overlap:   true,
			wantError: "ambiguous rule feature",
		},
		{
			name:      "id key collision",
			config:    `{"features":["` + featureA + `"]}`,
			collision: true,
			wantError: "ambiguous rule feature",
		},
		{
			name:      "non-array",
			config:    `{"features":"a"}`,
			wantError: "invalid rule features",
		},
		{
			name:      "malformed object ID",
			config:    `{"features":[{"id":"a"}]}`,
			wantError: "invalid rule feature reference",
		},
		{
			name:      "empty reference",
			config:    `{"features":[""]}`,
			wantError: "invalid rule feature reference",
		},
		{
			name:      "normalization prerequisite",
			config:    `{"type":"invoice.created","data":{"features":["a"]}}`,
			wantError: "expected normalized flat rule config",
		},
	}

	for _, eventType := range []string{
		"entitlements.balance.threshold", "v1.entitlement.balance_threshold.triggered",
		"entitlements.reset", "v1.entitlement.usage_period.reset",
	} {
		t.Run(eventType, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// Given historical configs, including extra JSON that must survive unchanged.
					tx, err := db.BeginTx(t.Context(), nil)
					require.NoError(t, err)
					defer func() { require.NoError(t, tx.Rollback()) }()

					namespace := ulid.Make().String()
					featureNamespace := namespace
					if tt.foreign {
						featureNamespace = ulid.Make().String()
					}

					createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
					retiredAt := createdAt.Add(2 * time.Hour)
					ruleUpdatedAt := createdAt.Add(time.Hour)
					if tt.ruleHour != nil {
						ruleUpdatedAt = createdAt.Add(time.Duration(*tt.ruleHour) * time.Hour)
					}

					_, err = tx.ExecContext(t.Context(), `INSERT INTO features (id, namespace, name, key, created_at, updated_at, archived_at, deleted_at)
                        VALUES ($1,$2,'A','a',$6,$6,CASE WHEN $3 THEN $7::timestamptz ELSE NULL END,CASE WHEN $8 THEN $7::timestamptz ELSE NULL END),
                               ($4,$5,'B','b',$6::timestamptz-interval '1 day',$6,NULL,NULL)`,
						featureA, featureNamespace, tt.archived || tt.reuse, featureB, namespace, createdAt, retiredAt, tt.featureDeleted)
					require.NoError(t, err)

					if tt.reuse || tt.collision {
						key := "a"
						replacementCreatedAt := retiredAt
						if tt.collision {
							key = featureA
						}

						if tt.overlap || tt.collision {
							replacementCreatedAt = createdAt
						}

						_, err = tx.ExecContext(t.Context(), `INSERT INTO features (id, namespace, name, key, created_at, updated_at)
                            VALUES ($1,$2,'Other',$3,$4,$4)`, featureC, namespace, key, replacementCreatedAt)
						require.NoError(t, err)
					}

					const precedingID = "00000000000000000000000001"
					const targetID = "00000000000000000000000002"
					_, err = tx.ExecContext(t.Context(), `INSERT INTO notification_rules (id, namespace, name, type, config, created_at, updated_at, disabled, deleted_at, annotations, metadata)
      VALUES ($1,$3,'preceding',$4,'{"features":["b"]}',now(),now(),false,NULL,NULL,NULL),
             ($2,$3,'target',$4,$5::jsonb || '{"untouched":9007199254740993,"thresholds":[{"type":"number","value":10}]}',$6::timestamptz-interval '1 hour',$6,$7,NULL,'{"preserve":true}','{"team":"billing"}')`, precedingID, targetID, namespace, eventType, tt.config, ruleUpdatedAt, !tt.enabled)
					require.NoError(t, err)

					if tt.deleteHour != nil {
						// Soft deletion updates both timestamps while preserving the config.
						deletedAt := createdAt.Add(time.Duration(*tt.deleteHour) * time.Hour)
						_, err = tx.ExecContext(t.Context(), `UPDATE notification_rules
                            SET updated_at = $2, deleted_at = $2, disabled = true WHERE id = $1`, targetID, deletedAt)
						require.NoError(t, err)
					}

					// Unknown and invoice contracts are not part of this migration.
					_, err = tx.ExecContext(t.Context(), `INSERT INTO notification_rules (id, namespace, name, type, config, created_at, updated_at, disabled)
      VALUES ($1,$3,'future','future.event','{"features":["unknown"]}',now(),now(),false),
             ($2,$3,'invoice','v1.billing.invoice.created','{}',now(),now(),false)`, ulid.Make().String(), ulid.Make().String(), namespace)
					require.NoError(t, err)

					// The read-only preflight must identify the same blockers as the migration.
					rows, err := tx.QueryContext(t.Context(), string(preflight))
					require.NoError(t, err)

					var issues []string

					for rows.Next() {
						var ns, id, kind string
						var updatedAt time.Time
						var reference, matches any
						var candidates pq.StringArray

						require.NoError(t, rows.Scan(&ns, &id, &updatedAt, &kind, &reference, &matches, &candidates))
						require.Equal(t, namespace, ns)
						require.Equal(t, targetID, id)

						if tt.wantCandidateIDs != nil {
							require.Equal(t, "ambiguous_feature", kind)
							require.Equal(t, tt.wantCandidateIDs, []string(candidates))
						}

						issues = append(issues, kind)
					}

					require.NoError(t, rows.Err())
					require.NoError(t, rows.Close())

					switch {
					case tt.dropAction != "":
						require.Equal(t, []string{tt.dropAction}, issues)
					case tt.wantError != "":
						require.Len(t, issues, 1)
					default:
						require.Empty(t, issues)
					}

					var before, attributes string
					require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT config::text, (to_jsonb(r)-'config')::text FROM notification_rules r WHERE id=$1`, targetID).Scan(&before, &attributes))

					_, err = tx.ExecContext(t.Context(), "SAVEPOINT migration")
					require.NoError(t, err)

					// When applying the migration, malformed or ambiguous data must reject the entire operation.
					_, err = tx.ExecContext(t.Context(), scripts["up"])
					if tt.wantError != "" {
						require.ErrorContains(t, err, tt.wantError)
						require.ErrorContains(t, err, namespace)
						require.ErrorContains(t, err, targetID)

						_, err = tx.ExecContext(t.Context(), "ROLLBACK TO SAVEPOINT migration")
						require.NoError(t, err)

						var actual, preceding string
						require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT config::text FROM notification_rules WHERE id=$1`, targetID).Scan(&actual))
						require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT config::text FROM notification_rules WHERE id=$1`, precedingID).Scan(&preceding))

						require.Equal(t, decodeJSONWithNumbers(t, before), decodeJSONWithNumbers(t, actual))
						require.Equal(t, decodeJSONWithNumbers(t, `{"features":["b"]}`), decodeJSONWithNumbers(t, preceding))

						return
					}

					require.NoError(t, err)

					// Then only feature references change, and repeat/rollback keep the resolved IDs.
					var actual, afterAttributes string
					require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT config::text, (to_jsonb(r)-'config')::text FROM notification_rules r WHERE id=$1`, targetID).Scan(&actual, &afterAttributes))

					var expectedJSON map[string]json.RawMessage
					require.NoError(t, json.Unmarshal([]byte(tt.expected), &expectedJSON))

					expectedJSON["untouched"] = json.RawMessage(`9007199254740993`)
					expectedJSON["thresholds"] = json.RawMessage(`[{"type":"number","value":10}]`)
					expectedBytes, err := json.Marshal(expectedJSON)
					require.NoError(t, err)

					require.Equal(t, decodeJSONWithNumbers(t, string(expectedBytes)), decodeJSONWithNumbers(t, actual))
					require.Equal(t, decodeJSONWithNumbers(t, attributes), decodeJSONWithNumbers(t, afterAttributes))

					for _, direction := range []string{"up", "down", "up"} {
						_, err = tx.ExecContext(t.Context(), scripts[direction])
						require.NoError(t, err)

						var repeated string
						require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT config::text FROM notification_rules WHERE id=$1`, targetID).Scan(&repeated))

						require.Equal(t, decodeJSONWithNumbers(t, actual), decodeJSONWithNumbers(t, repeated))

						// Resolved ID objects must not become ambiguous on a later preflight.
						rows, err := tx.QueryContext(t.Context(), string(preflight))
						require.NoError(t, err)

						require.False(t, rows.Next())
						require.NoError(t, rows.Err())
						require.NoError(t, rows.Close())
					}
				})
			}
		})
	}
}

// decodeJSONWithNumbers preserves exact JSON numbers when comparing migrated documents.
func decodeJSONWithNumbers(t *testing.T, data string) any {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()

	var decoded any
	require.NoError(t, decoder.Decode(&decoded))

	return decoded
}
