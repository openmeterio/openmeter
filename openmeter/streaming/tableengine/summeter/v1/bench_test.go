package summeterv1

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	pmentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	chstream "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// benchManageService is a minimal in-memory implementation of meter.ManageService
// sufficient for Engine.persistState calls in benchmarks.
type benchManageService struct {
	last meter.Meter
}

func (b *benchManageService) ListMeters(ctx context.Context, params meter.ListMetersParams) (res pagination.Result[meter.Meter], err error) {
	return res, fmt.Errorf("not implemented")
}

func (b *benchManageService) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	return meter.Meter{}, fmt.Errorf("not implemented")
}

func (b *benchManageService) CreateMeter(ctx context.Context, input meter.CreateMeterInput) (meter.Meter, error) {
	return meter.Meter{}, fmt.Errorf("not implemented")
}

func (b *benchManageService) UpdateMeter(ctx context.Context, input meter.UpdateMeterInput) (meter.Meter, error) {
	return meter.Meter{}, fmt.Errorf("not implemented")
}

func (b *benchManageService) DeleteMeter(ctx context.Context, input meter.DeleteMeterInput) error {
	return fmt.Errorf("not implemented")
}

func (b *benchManageService) UpdateTableEngine(ctx context.Context, m meter.Meter) error {
	b.last = m
	return nil
}

func (b *benchManageService) RegisterPreUpdateMeterHook(hook meter.PreUpdateMeterHook) error {
	return nil
}

// noop progress manager for the streaming connector
type noopProgressManager struct{}

func (n noopProgressManager) GetProgress(ctx context.Context, input pmentity.GetProgressInput) (*pmentity.Progress, error) {
	return nil, nil
}

func (n noopProgressManager) UpsertProgress(ctx context.Context, input pmentity.UpsertProgressInput) error {
	return nil
}

// Helper to create a large JSON payload of approx targetBytes with required fields.
func makeEventJSON(a, b string, amount float64, targetBytes int) string {
	base := fmt.Sprintf(`{"a":"%s","b":"%s","amount":%g`, a, b, amount)
	// leave room for closing brace and pad field scaffolding
	remaining := targetBytes - len(base) - len(`,"pad":""}`) - 1
	if remaining < 0 {
		remaining = 0
	}
	pad := strings.Repeat("x", remaining)
	return base + `,"pad":"` + pad + `"}`
}

func createCHConnFromEnv(b *testing.B) driver.Conn {
	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		b.Skip("TEST_CLICKHOUSE_DSN is not set; skipping benchmark")
	}
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		b.Fatalf("parse DSN: %v", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		b.Fatalf("open clickhouse: %v", err)
	}
	return conn
}

func BenchmarkQueryMeterComparison(b *testing.B) {
	ctx := context.Background()
	ch := createCHConnFromEnv(b)

	// payloadBytesCandidates := []int{512, 2048, 4096, 8192, 16384}
	payloadBytesCandidates := []int{0, 400}
	for _, payloadBytes := range payloadBytesCandidates {

		// Unique DB for benchmark run
		dbName := fmt.Sprintf("bench_%d", time.Now().UnixNano())
		if err := ch.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName)); err != nil {
			b.Fatalf("create db: %v", err)
		}
		fmt.Printf("database: %s\n", dbName)
		fmt.Printf("events table: %s.%s\n", dbName, "om_events")

		// Create events table via connector DDL
		if err := ch.Exec(ctx, chstream.CreateEventsTableSQL(dbName, "om_events")); err != nil {
			b.Fatalf("create events table: %v", err)
		}

		// Create numeric meter table
		engine := Engine{
			logger:     slog.Default(),
			database:   dbName,
			clickhouse: ch,
		}
		if err := ch.Exec(ctx, engine.CreateTableSQL()); err != nil {
			b.Fatalf("create meter table: %v", err)
		}

		// Seed 1,000,000 events (2KB each) across 100 subjects
		const (
			totalEvents   = 1_000_000
			numSubjects   = 100
			batchSize     = 10_000
			namespace     = "ns-bench"
			eventType     = "evt.bench"
			eventsTable   = "om_events"
			subjectPrefix = "subj-"
		)

		connector, err := chstream.New(ctx, chstream.Config{
			Logger:           slog.Default(),
			ClickHouse:       ch,
			Database:         dbName,
			EventsTableName:  eventsTable,
			ProgressManager:  noopProgressManager{},
			SkipCreateTables: true,
		})
		if err != nil {
			b.Fatalf("create connector: %v", err)
		}

		startTime := time.Now().UTC().Add(-1 * time.Hour)
		for offset := 0; offset < totalEvents; offset += batchSize {
			limit := min(totalEvents-offset, batchSize)
			events := make([]streaming.RawEvent, 0, limit)
			for i := 0; i < limit; i++ {
				idx := offset + i
				subj := fmt.Sprintf("%s%d", subjectPrefix, idx%numSubjects)
				// alternate group-bys (A0..A9) and (B0..B9)
				a := fmt.Sprintf("A%d", idx%10)
				bv := fmt.Sprintf("B%d", (idx/10)%10)
				amt := float64((idx%100)+1) / 10.0
				t := startTime.Add(time.Duration(idx%3600) * time.Second)
				data := makeEventJSON(a, bv, amt, payloadBytes)
				id := fmt.Sprintf("id-%d", idx)
				events = append(events, streaming.RawEvent{
					Namespace:  namespace,
					ID:         id,
					Type:       eventType,
					Source:     "src",
					Subject:    subj,
					Time:       t,
					Data:       data,
					IngestedAt: t,
					StoredAt:   t,
					StoreRowID: id,
				})
			}
			if err := connector.BatchInsert(ctx, events); err != nil {
				b.Fatalf("batch insert: %v", err)
			}
		}

		// Prepare meter and backfill numeric meter table via InsertFromEvents
		m := meter.Meter{
			Aggregation:   meter.MeterAggregationSum,
			EventType:     eventType,
			ValueProperty: lo.ToPtr("$.amount"),
			GroupBy: map[string]string{
				"g1": "$.a",
				"g2": "$.b",
			},
		}
		m.ManagedResource.ID = "meter-bench"
		m.ManagedResource.NamespacedModel.Namespace = namespace
		m.TableEngine = &meter.MeterTableEngine{
			Engine: TableEngineName,
			Status: meter.MeterTableEngineStateActive,
			State:  "{}",
		}
		engine.meterService = &benchManageService{}

		// Determine min stored_at and insert from events across full range
		minStoredAt, err := engine.MinEventsStoredAt(ctx, eventsTable, namespace, eventType)
		if err != nil {
			b.Fatalf("min stored_at: %v", err)
		}
		if minStoredAt == nil {
			b.Fatalf("no events found to backfill")
		}
		period := timeutil.ClosedPeriod{
			From: minStoredAt.UTC().Truncate(time.Second),
			To:   startTime.Add(2 * time.Hour).UTC().Truncate(time.Second),
		}
		if err := engine.InsertFromEvents(ctx, eventsTable, m, period); err != nil {
			b.Fatalf("insert from events: %v", err)
		}

		// Prepare common query params: choose 10 subjects, group-by subject and filter on g1
		subjects := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			subjects = append(subjects, fmt.Sprintf("%s%d", subjectPrefix, i))
		}
		params := streaming.QueryParams{
			GroupBy:       []string{"subject"},
			FilterSubject: subjects,
			FilterGroupBy: map[string]filter.FilterString{
				"g1": {Eq: lo.ToPtr("A1")},
			},
		}
		from := startTime
		to := startTime.Add(1 * time.Hour)
		params.From = &from
		params.To = &to

		time.Sleep(5 * time.Second)
		assertMeterCount(b, ch, dbName, namespace, m.ID, totalEvents)

		// Benchmark A: table engine query (numeric_meter_v1) using summeter/v1 queryMeter
		b.Run(fmt.Sprintf("tableengine_queryMeter_%d", payloadBytes), func(sb *testing.B) {
			qm := queryMeter{
				Database:     dbName,
				SumTableName: TableName,
				Namespace:    namespace,
				Meter:        m,
				QueryParams:  params,
			}
			// Build once to detect any errors upfront
			_, _, err := qm.ToSQL()
			if err != nil {
				sb.Fatalf("build sql: %v", err)
			}
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				// Deterministically select one subject per-iteration
				selected := subjects[i%len(subjects)]
				qm.QueryParams.FilterSubject = []string{selected}
				sql, args, err := qm.ToSQL()
				if err != nil {
					sb.Fatalf("build sql: %v", err)
				}
				rows, err := ch.Query(ctx, sql, args...)
				if err != nil {
					sb.Fatalf("clickhouse query: %v", err)
				}
				_, err = qm.ScanRows(rows)
				rows.Close()
				if err != nil {
					sb.Fatalf("scan rows: %v", err)
				}
			}
		})

		// Benchmark B: streaming connector query meter (events table)
		b.Run(fmt.Sprintf("streaming_queryMeter_%d", payloadBytes), func(sb *testing.B) {
			// Copy meter and clear table engine to force streaming query path
			m2 := m
			m2.TableEngine = nil
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				// Deterministically select one subject per-iteration
				selected := subjects[i%len(subjects)]
				iterParams := params
				iterParams.FilterSubject = []string{selected}
				_, err := connector.QueryMeter(ctx, namespace, m2, iterParams)
				if err != nil {
					sb.Fatalf("connector QueryMeter: %v", err)
				}
			}
		})

		// Benchmark C table engine query (numeric_meter_v1) using summeter/v1 queryMeter, no group by
		b.Run(fmt.Sprintf("tableengine_queryMeter_no_group_by_%d", payloadBytes), func(sb *testing.B) {
			qm := queryMeter{
				Database:     dbName,
				SumTableName: TableName,
				Namespace:    namespace,
				Meter:        m,
				QueryParams:  params,
			}
			// Build once to detect any errors upfront
			_, _, err := qm.ToSQL()
			if err != nil {
				sb.Fatalf("build sql: %v", err)
			}
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				// Deterministically select one subject per-iteration
				selected := subjects[i%len(subjects)]
				qm.QueryParams.FilterSubject = []string{selected}
				qm.QueryParams.GroupBy = []string{}
				sql, args, err := qm.ToSQL()
				if err != nil {
					sb.Fatalf("build sql: %v", err)
				}
				rows, err := ch.Query(ctx, sql, args...)
				if err != nil {
					sb.Fatalf("clickhouse query: %v", err)
				}
				_, err = qm.ScanRows(rows)
				rows.Close()
				if err != nil {
					sb.Fatalf("scan rows: %v", err)
				}
			}
		})

		// Benchmark D: streaming connector query meter (events table) no group by
		b.Run(fmt.Sprintf("streaming_queryMeter_no_group_by_%d", payloadBytes), func(sb *testing.B) {
			// Copy meter and clear table engine to force streaming query path
			m2 := m
			m2.TableEngine = nil
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				// Deterministically select one subject per-iteration
				selected := subjects[i%len(subjects)]
				iterParams := params
				iterParams.FilterSubject = []string{selected}
				iterParams.GroupBy = []string{}
				_, err := connector.QueryMeter(ctx, namespace, m2, iterParams)
				if err != nil {
					sb.Fatalf("connector QueryMeter: %v", err)
				}
			}
		})

		// Benchmark E table engine query (numeric_meter_v1) using summeter/v1 queryMeter, no group by
		b.Run(fmt.Sprintf("tableengine_queryMeter_no_group_by_float64_%d", payloadBytes), func(sb *testing.B) {
			qm := queryMeter{
				Database:      dbName,
				SumTableName:  TableName,
				Namespace:     namespace,
				Meter:         m,
				QueryParams:   params,
				UseFloatValue: true,
			}
			// Build once to detect any errors upfront
			_, _, err := qm.ToSQL()
			if err != nil {
				sb.Fatalf("build sql: %v", err)
			}
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				// Deterministically select one subject per-iteration
				selected := subjects[i%len(subjects)]
				qm.QueryParams.FilterSubject = []string{selected}
				qm.QueryParams.GroupBy = []string{}
				sql, args, err := qm.ToSQL()
				if err != nil {
					sb.Fatalf("build sql: %v", err)
				}
				rows, err := ch.Query(ctx, sql, args...)
				if err != nil {
					sb.Fatalf("clickhouse query: %v", err)
				}
				_, err = qm.ScanRows(rows)
				rows.Close()
				if err != nil {
					sb.Fatalf("scan rows: %v", err)
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// assertMeterCount validates the number of records for a meter in numeric_meter_v1.
func assertMeterCount(b *testing.B, ch driver.Conn, dbName, namespace, meterID string, expect uint64) {
	sql := fmt.Sprintf("SELECT count() FROM %s.%s WHERE namespace = ? AND meter_id = ?", dbName, TableName)
	rows, err := ch.Query(context.Background(), sql, namespace, meterID)
	if err != nil {
		b.Fatalf("count meter rows: %v", err)
	}
	defer rows.Close()
	var cnt uint64
	if rows.Next() {
		if err := rows.Scan(&cnt); err != nil {
			b.Fatalf("scan meter count: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		b.Fatalf("rows error: %v", err)
	}
	if cnt != expect {
		b.Fatalf("meter row count mismatch: got %d, expect %d", cnt, expect)
	}
}
