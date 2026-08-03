package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

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

		service := NewManage(meterAdapter, eventbus.NewMock(t), nil, nil)

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
		}, nil, nil)

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
