package clickhouse

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	eventsTableName = "events"
	namespace       = "test-namespace"
)

type ConnectorTestSuite struct {
	CHTestSuite
	Connector *Connector
}

func (s *ConnectorTestSuite) SetupTest() {
	if s.T().Skipped() {
		return
	}

	s.CHTestSuite.SetupTest()

	connector, err := New(s.T().Context(), Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		ProgressManager: progressmanager.NewMockProgressManager(),
		AsyncInsert:     false,
	})
	s.NoError(err, "failed to create connector")
	s.Connector = connector
}

func (s *ConnectorTestSuite) TearDownTest() {
	if s.T().Skipped() {
		return
	}

	s.CHTestSuite.TearDownTest()

	s.Connector = nil
}

func (s *ConnectorTestSuite) TestConnectorQueryMeter() {
	t := s.T()
	ctx := t.Context()
	now := time.Now().UTC()
	from := now.Add(-time.Hour)
	to := now
	eventTime := now.Add(-time.Minute)
	subject := "test-subject"
	eventType := "test-event"

	err := s.Connector.BatchInsert(ctx, []streaming.RawEvent{
		{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Time:       eventTime,
			Type:       eventType,
			Source:     "test-source",
			Subject:    subject,
			Data:       `{"value": 123, "name": 1}`,
			IngestedAt: now,
			StoredAt:   now,
		},
		{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Time:       eventTime.Add(time.Second),
			Type:       eventType,
			Source:     "test-source",
			Subject:    subject,
			Data:       `{"value": "0.4567890123456789", "name": "a"}`,
			IngestedAt: now,
			StoredAt:   now,
		},
		{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Time:       eventTime.Add(2 * time.Second),
			Type:       eventType,
			Source:     "test-source",
			Subject:    subject,
			Data:       `{"value": null,"name": null}`,
			IngestedAt: now,
			StoredAt:   now,
		},
	})
	s.NoError(err)

	tests := []struct {
		meterAggregation       meter.MeterAggregation
		valueProperty          *string
		enableDecimalPrecision bool
		from                   time.Time
		to                     time.Time
		wantValue              *float64
	}{
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(123.45678901234568),
		},
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(123.45678901234568),
		},
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(3)),
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(3)),
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              lo.ToPtr(float64(0)),
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              lo.ToPtr(float64(0)),
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(61.72839450617284),
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(61.72839450617284),
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(0.4567890123456789),
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(0.4567890123456789),
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(123)),
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(123)),
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(2)),
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(float64(2)),
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              lo.ToPtr(float64(0)),
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              lo.ToPtr(float64(0)),
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(0.4567890123456789),
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   from,
			to:                     to,
			wantValue:              lo.ToPtr(0.4567890123456789),
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			from:                   to,
			to:                     to.Add(time.Hour),
			wantValue:              nil,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("meter aggregation: %s, enable decimal precision: %t", tt.meterAggregation, tt.enableDecimalPrecision), func(t *testing.T) {
			m := meter.Meter{
				ManagedResource: models.ManagedResource{
					ID:   ulid.Make().String(),
					Name: "test-meter",
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				Key:           "test-meter",
				EventType:     eventType,
				ValueProperty: tt.valueProperty,
				Aggregation:   tt.meterAggregation,
			}
			err = m.Validate()
			s.NoError(err)

			s.Connector.config.EnableDecimalPrecision = tt.enableDecimalPrecision

			rows, err := s.Connector.QueryMeter(ctx, namespace, m, streaming.QueryParams{
				From: &tt.from,
				To:   &tt.to,
			})

			s.NoError(err)

			if tt.wantValue == nil {
				s.Len(rows, 0)
				return
			}

			s.Equal([]meter.MeterQueryRow{
				{
					WindowStart: tt.from,
					WindowEnd:   tt.to,
					Value:       *tt.wantValue,
					GroupBy:     map[string]*string{},
				},
			}, rows)
		})
	}
}

func TestConnector(t *testing.T) {
	suite.Run(t, new(ConnectorTestSuite))
}
