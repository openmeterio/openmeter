package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestCustomerLifecycleThroughEventBus(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		rollback     bool
		rejectInsert bool
	}{
		{name: "committed customer is readable when delivered"},
		{name: "outer rollback discards customer and event", rollback: true},
		{name: "outbox insert failure rolls back customer", rejectInsert: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Given the production customer service and CloudEvent pipeline sharing a database.
			raw := newRecordingPublisher()
			publisher, client := newTestPublisher(t, raw)
			logger := testutils.NewDiscardLogger(t)
			bus, err := eventbus.New(eventbus.Options{
				Publisher: publisher,
				TopicMapping: eventbus.TopicMapping{
					SystemEventsTopic: testTopic, IngestEventsTopic: "ingest", BalanceWorkerEventsTopic: "balance",
				},
				Logger:                 logger,
				MarshalerTransformFunc: kafka.AddPartitionKeyFromSubject,
			})
			require.NoError(t, err)
			adapter, err := customeradapter.New(customeradapter.Config{Client: client, Logger: logger})
			require.NoError(t, err)
			service, err := customerservice.New(customerservice.Config{Adapter: adapter, Publisher: bus})
			require.NoError(t, err)
			if scenario.rejectInsert {
				_, err := client.ExecContext(t.Context(), "ALTER TABLE event_outboxes ADD CONSTRAINT reject_test_event CHECK (topic <> 'system-events')")
				require.NoError(t, err)
			}
			var created *customer.Customer
			rollbackErr := errors.New("outer operation failed")

			// When a nested customer operation publishes before its outer transaction finishes.
			err = transaction.RunWithNoValue(t.Context(), adapter, func(ctx context.Context) error {
				var err error
				created, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
					Namespace:      "outbox-test",
					CustomerMutate: customer.CustomerMutate{Name: "Customer", Key: lo.ToPtr("customer")},
				})
				if err != nil {
					return err
				}
				noAttempt(t, raw)
				if scenario.rollback {
					return rollbackErr
				}
				return nil
			})

			// Then the event and customer have the same outcome, with the existing wire format/key.
			if scenario.rollback || scenario.rejectInsert {
				if scenario.rollback {
					require.ErrorIs(t, err, rollbackErr)
				} else {
					require.ErrorContains(t, err, "enqueue system event")
				}
				noAttempt(t, raw)
				count, err := client.Customer.Query().Count(t.Context())
				require.NoError(t, err)
				require.Zero(t, count)
			} else {
				require.NoError(t, err)
				attempt := nextAttempt(t, raw)
				msg := message.NewMessage(attempt.id, attempt.payload)
				msg.Metadata = attempt.metadata
				var event customer.CustomerCreateEvent
				require.NoError(t, bus.Marshaler().Unmarshal(msg, &event))
				require.Equal(t, created.ID, event.Customer.ID)
				require.Equal(t, event.EventName(), attempt.metadata.Get("ce_type"))
				require.Equal(t, metadata.ComposeResourcePath(created.Namespace, metadata.EntityCustomer, created.ID), attempt.metadata.Get(kafka.PartitionKeyMetadataKey))
				stored, err := client.Customer.Get(t.Context(), created.ID)
				require.NoError(t, err)
				require.Equal(t, created.Name, stored.Name)
			}
			eventuallyRowCount(t, client, 0)
		})
	}
}
