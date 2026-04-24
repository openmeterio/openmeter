# message

<!-- archie:ai-start -->

> Provides the Batch type alias (= service.MessageBatch) and Transaction primitive for associating a message batch with its acknowledgement channel. Used internally to route batches through pipeline stages with delivery guarantees.

## Patterns

**Type alias for Benthos MessageBatch** — Batch is a type alias (not a newtype) for service.MessageBatch. All Benthos MessageBatch methods are available directly on Batch. (`type Batch = service.MessageBatch`)
**Transaction carries Payload + ack mechanism** — Transaction holds a Batch and either a responseChan chan<-error or a responseFunc. Call Ack(ctx, err) to acknowledge; nil err = success, non-nil = retry upstream. (`tx := message.NewTransactionFunc(batch, func(ctx context.Context, err error) error { ... })
tx.Ack(ctx, nil)`)
**WithContext for cancellation** — WithContext returns a shallow copy of the Transaction with a new ctx. Receivers should honor t.Context() cancellation but must still call Ack even when canceled. (`tx = tx.WithContext(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `batch.go` | Single type alias for service.MessageBatch | This is an alias (=), not a defined type — do not add methods to Batch directly |
| `transaction.go` | Transaction struct with NewTransaction (chan-based) and NewTransactionFunc (func-based) constructors | Exactly one of responseChan or responseFunc will be non-nil; Ack branches on responseFunc != nil first |

## Anti-Patterns

- Not calling Ack after processing — the upstream source will stall waiting for acknowledgement
- Calling Ack multiple times on the same Transaction — chan send and responseFunc are not idempotent
- Ignoring t.Context() cancellation in a long reconnect loop — can block shutdown

## Decisions

- **Two ack paths: chan<-error vs func(context.Context, error) error** — Chan-based ack is simpler for single-consumer paths; func-based ack supports async/contextual delivery confirmation needed by more complex output brokers
- **Transaction is its own concept rather than embedding service.Message** — Batch expansion (splitting one batch into derived batches) breaks direct message-level ack; Transaction decouples ack from the batch contents so derived transactions can be tracked independently

## Example: Create and acknowledge a function-based transaction

```
import (
	"context"
	"github.com/openmeterio/openmeter/collector/benthos/internal/message"
)

func process(batch message.Batch) error {
	tx := message.NewTransactionFunc(batch, func(ctx context.Context, err error) error {
		// propagate ack/nack to upstream source
		return err
	})
	// ... pipeline stages ...
	return tx.Ack(context.Background(), nil)
}
```

<!-- archie:ai-end -->
