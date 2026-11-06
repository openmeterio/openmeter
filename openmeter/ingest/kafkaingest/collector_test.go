package kafkaingest

import (
	"context"
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func TestCollectorWaitForDelivery(t *testing.T) {
	t.Run("returns delivery error", func(t *testing.T) {
		c := Collector{}

		delivery := make(chan kafka.Event, 1)
		delivery <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{Error: errors.New("Local: Message timed out")},
		}
		close(delivery)

		if err := c.waitForDelivery(t.Context(), delivery); err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("returns nil on successful delivery", func(t *testing.T) {
		c := Collector{}

		delivery := make(chan kafka.Event, 1)
		delivery <- &kafka.Message{}
		close(delivery)

		if err := c.waitForDelivery(t.Context(), delivery); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("rejects unexpected event type", func(t *testing.T) {
		c := Collector{}

		delivery := make(chan kafka.Event, 1)
		delivery <- kafka.Error{}
		close(delivery)

		if err := c.waitForDelivery(t.Context(), delivery); err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("returns context error when canceled", func(t *testing.T) {
		c := Collector{}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		delivery := make(chan kafka.Event, 1)
		if err := c.waitForDelivery(ctx, delivery); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	})
}
