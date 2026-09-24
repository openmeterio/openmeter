package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	DB               *db.Client
	Publisher        message.Publisher
	Topic            string
	Logger           *slog.Logger
	DrainLimit       int
	DrainTimeout     time.Duration
	DrainConcurrency int
	RetryInterval    time.Duration
}

func (c Config) Validate() error {
	var errs []error
	if c.DB == nil {
		errs = append(errs, errors.New("database is required"))
	}
	if c.Publisher == nil {
		errs = append(errs, errors.New("publisher is required"))
	}
	if c.Topic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}
	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}
	if c.DrainLimit <= 0 {
		errs = append(errs, errors.New("drain limit must be greater than 0"))
	}
	if c.DrainTimeout <= 0 {
		errs = append(errs, errors.New("drain timeout must be greater than 0"))
	}
	if c.DrainConcurrency <= 0 {
		errs = append(errs, errors.New("drain concurrency must be greater than 0"))
	}
	if c.RetryInterval <= 0 {
		errs = append(errs, errors.New("retry interval must be greater than 0"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Publisher persists system events in the caller's transaction. Each successful
// commit wakes a bounded drain of the shared queue; a periodic tick retries idle work.
type Publisher struct {
	cfg     Config
	ctx     context.Context
	cancel  context.CancelFunc
	wake    chan struct{}
	workers sync.WaitGroup
	mu      sync.RWMutex
	closed  bool
}

var _ message.Publisher = (*Publisher)(nil)

func NewPublisher(ctx context.Context, cfg Config) (*Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	p := &Publisher{cfg: cfg, ctx: ctx, cancel: cancel, wake: make(chan struct{}, cfg.DrainConcurrency)}
	for range cfg.DrainConcurrency {
		p.workers.Add(1)
		go p.run()
	}
	return p, nil
}

func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return errors.New("outbox publisher is closed")
	}
	if topic != p.cfg.Topic {
		return p.cfg.Publisher.Publish(topic, messages...)
	}
	for _, msg := range messages {
		if msg == nil {
			return errors.New("outbox message is nil")
		}
		deliveryKey := msg.Metadata.Get(kafka.PartitionKeyMetadataKey)
		// Run joins the caller's transaction, or commits a standalone enqueue for
		// system events whose producer does not have a database transaction.
		err := transaction.RunWithNoValue(msg.Context(), enttx.NewCreator(p.cfg.DB), func(ctx context.Context) error {
			tx, err := entutils.GetDriverFromContext(ctx)
			if err != nil {
				return err
			}
			client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
			_, err = client.EventOutbox.Create().
				SetMessageID(msg.UUID).
				SetTopic(topic).
				SetDeliveryKey(deliveryKey).
				SetPayload(msg.Payload).
				SetMetadata(map[string]string(msg.Metadata)).Save(ctx)
			if err != nil {
				return fmt.Errorf("enqueue system event: %w", err)
			}
			return tx.AfterCommit(p.signal)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Publisher) signal() {
	for range p.cfg.DrainConcurrency {
		select {
		case p.wake <- struct{}{}:
		default:
		}
	}
}

func (p *Publisher) run() {
	defer p.workers.Done()
	ticker := time.NewTicker(p.cfg.RetryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.wake:
		case <-ticker.C:
		}
		ctx, cancel := context.WithTimeout(p.ctx, p.cfg.DrainTimeout)
		err := p.drain(ctx)
		cancel()
		if err != nil && p.ctx.Err() == nil {
			p.cfg.Logger.WarnContext(p.ctx, "system event delivery deferred until the next publish or retry tick", "error", err)
		}
	}
}

func (p *Publisher) drain(ctx context.Context) error {
	var failedKeys []string
	var errs []error
	delivered := false
	for range p.cfg.DrainLimit {
		var deliveryKey string
		var claimed bool
		sent, err := transaction.Run(ctx, enttx.NewCreator(p.cfg.DB), func(ctx context.Context) (bool, error) {
			tx, err := entutils.GetDriverFromContext(ctx)
			if err != nil {
				return false, err
			}
			client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
			// Only a key's oldest committed row is eligible. A locked or failed
			// head remains a barrier for later rows with the same key.
			query := pendingQuery{Topic: p.cfg.Topic, ExcludedKeys: failedKeys}
			row, err := client.EventOutbox.Query().
				Where(query.Apply).
				Order(eventoutbox.ByID()).
				ForUpdate(sql.WithLockAction(sql.SkipLocked)).First(ctx)
			if db.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			deliveryKey = row.DeliveryKey
			claimed = true
			msg := message.NewMessage(row.MessageID, row.Payload)
			msg.Metadata = message.Metadata(row.Metadata)
			msg.SetContext(ctx)
			if err := p.cfg.Publisher.Publish(row.Topic, msg); err != nil {
				return false, fmt.Errorf("publish pending system event: %w", err)
			}
			// A crash after broker acknowledgment but before this transaction
			// commits leaves the same message ID pending for at-least-once delivery.
			if err := client.EventOutbox.DeleteOneID(row.ID).Exec(ctx); err != nil {
				return false, fmt.Errorf("remove delivered system event: %w", err)
			}
			return true, nil
		})
		if err != nil {
			errs = append(errs, err)
			if !claimed || ctx.Err() != nil {
				return errors.Join(errs...)
			}
			// Retry this key on a later wake while unrelated keys can progress.
			failedKeys = append(failedKeys, deliveryKey)
			continue
		}
		if !sent {
			return errors.Join(errs...)
		}
		delivered = true
	}
	// A large committed batch may have coalesced its notifications. Continue
	// productive work without introducing a timer or retrying an all-failed queue.
	if delivered {
		p.signal()
	}
	return errors.Join(errs...)
}

// Close stops delivery attempts. Pending events remain durable for a future
// publish, and the caller retains ownership of the underlying Kafka publisher.
func (p *Publisher) Close() error {
	p.mu.Lock()
	p.closed = true
	p.cancel()
	p.mu.Unlock()
	p.workers.Wait()
	return nil
}
