# message

<!-- archie:ai-start -->

> Provides the Batch type alias (= service.MessageBatch) and the Transaction primitive that associates a batch with its acknowledgement mechanism, enabling delivery guarantees as batches flow through pipeline stages.

## Patterns

**Batch is a type alias, not a newtype** — type Batch = service.MessageBatch — all Benthos MessageBatch methods are available directly. Do not add methods to Batch; it is purely a transparent alias. (`type Batch = service.MessageBatch`)
**Transaction pairs Payload with ack mechanism** — Transaction holds a Batch and either a responseChan chan<-error or a responseFunc. Use NewTransaction for chan-based ack, NewTransactionFunc for func-based ack. Call Ack(ctx, nil) for success, Ack(ctx, err) to trigger retry upstream. (`tx := message.NewTransactionFunc(batch, func(ctx context.Context, err error) error { return err })
return tx.Ack(context.Background(), nil)`)
**WithContext for cancellation propagation** — WithContext returns a shallow copy of the Transaction with the new ctx. Receivers should honor t.Context() cancellation but must still call Ack even when canceled, passing t.Context().Err(). (`tx = tx.WithContext(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `batch.go` | Single type alias for service.MessageBatch. | This is an alias (=), not a defined type — do not add methods to Batch directly. |
| `transaction.go` | Transaction struct with NewTransaction (chan-based) and NewTransactionFunc (func-based) constructors plus Ack and WithContext. | Exactly one of responseChan or responseFunc is non-nil; Ack branches on responseFunc != nil first. Calling Ack multiple times is not idempotent. |

## Anti-Patterns

- Not calling Ack after processing — the upstream source will stall waiting for acknowledgement
- Calling Ack multiple times on the same Transaction — chan send and responseFunc are not idempotent
- Ignoring t.Context() cancellation in a long reconnect loop — can block shutdown
- Adding methods to Batch — it is a type alias and must remain a transparent pass-through

## Decisions

- **Two ack paths: chan<-error vs func(context.Context, error) error** — Chan-based ack is simpler for single-consumer paths; func-based ack supports async/contextual delivery confirmation needed by more complex output brokers
- **Transaction is its own concept rather than embedding service.Message** — Batch expansion (splitting one batch into derived batches) breaks direct message-level ack; Transaction decouples ack from batch contents so derivative transactions can be tracked independently

## Example: Create and acknowledge a function-based transaction

```
import (
	"context"
	"github.com/openmeterio/openmeter/collector/benthos/internal/message"
)

func process(batch message.Batch) error {
	tx := message.NewTransactionFunc(batch, func(ctx context.Context, err error) error {
		return err
	})
	// ... pipeline stages ...
	return tx.Ack(context.Background(), nil)
}
```

<!-- archie:ai-end -->
