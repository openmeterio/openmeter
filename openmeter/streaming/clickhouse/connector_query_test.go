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
			Data:       `{"value": 123,"name": "a"}`,
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
			Data:       `{"value": "0.4567890123456789","name": "a"}`,
			IngestedAt: now,
			StoredAt:   now,
		},
	})
	s.NoError(err)

	tests := []struct {
		meterAggregation       meter.MeterAggregation
		valueProperty          *string
		enableDecimalPrecision bool
		wantValue              float64
	}{
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			wantValue:              123.45678901234568,
		},
		{
			meterAggregation:       meter.MeterAggregationSum,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			wantValue:              123.45678901234568,
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: false,
			wantValue:              2,
		},
		{
			meterAggregation:       meter.MeterAggregationCount,
			enableDecimalPrecision: true,
			wantValue:              2,
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			wantValue:              61.72839450617284,
		},
		{
			meterAggregation:       meter.MeterAggregationAvg,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			wantValue:              61.72839450617284,
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			wantValue:              0.4567890123456789,
		},
		{
			meterAggregation:       meter.MeterAggregationMin,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			wantValue:              0.4567890123456789,
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			wantValue:              123,
		},
		{
			meterAggregation:       meter.MeterAggregationMax,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			wantValue:              123,
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: false,
			wantValue:              1,
		},
		{
			meterAggregation:       meter.MeterAggregationUniqueCount,
			valueProperty:          lo.ToPtr("$.name"),
			enableDecimalPrecision: true,
			wantValue:              1,
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: false,
			wantValue:              0.4567890123456789,
		},
		{
			meterAggregation:       meter.MeterAggregationLatest,
			valueProperty:          lo.ToPtr("$.value"),
			enableDecimalPrecision: true,
			wantValue:              0.4567890123456789,
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
				From: &from,
				To:   &to,
			})

			s.NoError(err)
			s.Equal([]meter.MeterQueryRow{
				{
					WindowStart: from,
					WindowEnd:   to,
					Value:       tt.wantValue,
					GroupBy:     map[string]*string{},
				},
			}, rows)
		})
	}
}

func TestConnector(t *testing.T) {
	suite.Run(t, new(ConnectorTestSuite))
}
