package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

type jsonPathValidatorFunc func(jsonPath string) (bool, error)

func (f jsonPathValidatorFunc) ValidateJSONPath(_ context.Context, jsonPath string) (bool, error) {
	return f(jsonPath)
}

type errorPublisher struct {
	eventbus.Publisher
	err error
}

func (p errorPublisher) Publish(context.Context, marshaler.Event) error {
	return p.err
}

func TestDeleteMeterTransaction(t *testing.T) {
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})

	meterAdapter, err := adapter.New(adapter.Config{
		Client: database.EntDriver.Client(),
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	t.Run("commits deletion after publishing the event", func(t *testing.T) {
		// Given a meter that has no active features or entitlements.
		namespace := ulid.Make().String()
		createdMeter, err := meterAdapter.CreateMeter(t.Context(), meter.CreateMeterInput{
			Namespace:   namespace,
			Name:        "meter to delete",
			Key:         "meter-to-delete",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "test.meter",
		})
		require.NoError(t, err)

		service := NewManage(meterAdapter, eventbus.NewMock(t), nil, nil, nil)

		// When the meter is deleted successfully.
		err = service.DeleteMeter(t.Context(), meter.DeleteMeterInput{
			Namespace: namespace,
			IDOrSlug:  createdMeter.ID,
		})

		// Then the transaction commits the soft deletion.
		require.NoError(t, err)
		deletedMeter, err := meterAdapter.GetMeterByIDOrSlug(t.Context(), meter.GetMeterInput{
			Namespace: namespace,
			IDOrSlug:  createdMeter.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, deletedMeter.DeletedAt)
	})

	t.Run("rolls back deletion when publishing fails", func(t *testing.T) {
		// Given a meter and a publisher that fails after the soft deletion.
		namespace := ulid.Make().String()
		createdMeter, err := meterAdapter.CreateMeter(t.Context(), meter.CreateMeterInput{
			Namespace:   namespace,
			Name:        "meter to preserve",
			Key:         "meter-to-preserve",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "test.meter",
		})
		require.NoError(t, err)

		publishErr := errors.New("publish failed")
		service := NewManage(meterAdapter, errorPublisher{
			Publisher: eventbus.NewMock(t),
			err:       publishErr,
		}, nil, nil, nil)

		// When deletion reaches event publication.
		err = service.DeleteMeter(t.Context(), meter.DeleteMeterInput{
			Namespace: namespace,
			IDOrSlug:  createdMeter.ID,
		})

		// Then the publication error rolls back the soft deletion.
		require.ErrorIs(t, err, publishErr)
		activeMeter, err := meterAdapter.GetMeterByIDOrSlug(t.Context(), meter.GetMeterInput{
			Namespace: namespace,
			IDOrSlug:  createdMeter.ID,
		})
		require.NoError(t, err)
		require.Nil(t, activeMeter.DeletedAt)
	})
}

func TestMeterJSONPathValidation(t *testing.T) {
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})

	meterAdapter, err := adapter.New(adapter.Config{
		Client: database.EntDriver.Client(),
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	// ClickHouse rejects hyphenated keys in dot notation; see TestConnectorValidateJSONPath.
	rejectHyphenatedKey := jsonPathValidatorFunc(func(jsonPath string) (bool, error) {
		return jsonPath != "$.transaction-count", nil
	})

	t.Run("create rejects a value property the event store cannot evaluate", func(t *testing.T) {
		// Given a value property that passes meter validation but not ClickHouse.
		namespace := ulid.Make().String()
		service := NewManage(meterAdapter, eventbus.NewMock(t), nil, nil, rejectHyphenatedKey)

		// When the meter is created.
		_, err := service.CreateMeter(t.Context(), meter.CreateMeterInput{
			Namespace:     namespace,
			Name:          "transactions",
			Key:           "transactions",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "transaction",
			ValueProperty: lo.ToPtr("$.transaction-count"),
		})

		// Then it is a validation error and no meter is stored.
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got %v", err)
		require.ErrorContains(t, err, "invalid value property JSONPath: $.transaction-count")

		_, err = meterAdapter.GetMeterByIDOrSlug(t.Context(), meter.GetMeterInput{
			Namespace: namespace,
			IDOrSlug:  "transactions",
		})
		require.True(t, meter.IsMeterNotFoundError(err), "expected not found, got %v", err)
	})

	t.Run("update rejects a group by the event store cannot evaluate", func(t *testing.T) {
		// Given an existing meter.
		namespace := ulid.Make().String()
		existing, err := meterAdapter.CreateMeter(t.Context(), meter.CreateMeterInput{
			Namespace:   namespace,
			Name:        "transactions",
			Key:         "transactions",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "transaction",
		})
		require.NoError(t, err)

		service := NewManage(meterAdapter, eventbus.NewMock(t), nil, nil, rejectHyphenatedKey)

		// When a group by with an unparseable path is added.
		_, err = service.UpdateMeter(t.Context(), meter.UpdateMeterInput{
			ID:      models.NamespacedID{Namespace: namespace, ID: existing.ID},
			Name:    existing.Name,
			GroupBy: map[string]string{"count": "$.transaction-count"},
		})

		// Then it is a validation error naming the group by.
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got %v", err)
		require.ErrorContains(t, err, "invalid group by count JSONPath: $.transaction-count")
	})

	t.Run("event store failures are not reported as validation errors", func(t *testing.T) {
		// Given an event store that cannot be reached.
		unreachable := errors.New("clickhouse unreachable")
		service := NewManage(meterAdapter, eventbus.NewMock(t), nil, nil, jsonPathValidatorFunc(func(string) (bool, error) {
			return false, unreachable
		}))

		// When a meter with a value property is created.
		_, err := service.CreateMeter(t.Context(), meter.CreateMeterInput{
			Namespace:     ulid.Make().String(),
			Name:          "transactions",
			Key:           "transactions",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "transaction",
			ValueProperty: lo.ToPtr("$.count"),
		})

		// Then the infrastructure error surfaces unchanged.
		require.ErrorIs(t, err, unreachable)
		require.False(t, models.IsGenericValidationError(err))
	})
}
