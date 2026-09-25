package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	DB        *db.Client
	Publisher message.Publisher

	// OutboxTopics lists topics persisted before delivery; all others publish directly.
	OutboxTopics     []string
	Logger           *slog.Logger
	DrainLimit       int
	DrainTimeout     time.Duration
	DrainConcurrency int
	RetryInterval    time.Duration
	MaxAttempts      int
}

func (c Config) Validate() error {
	var errs []error
	if c.DB == nil {
		errs = append(errs, errors.New("database is required"))
	}
	if c.Publisher == nil {
		errs = append(errs, errors.New("publisher is required"))
	}
	if len(c.OutboxTopics) == 0 {
		errs = append(errs, errors.New("at least one outbox topic is required"))
	}
	if slices.Contains(c.OutboxTopics, "") {
		errs = append(errs, errors.New("outbox topics must not be empty"))
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
	if c.MaxAttempts <= 0 {
		errs = append(errs, errors.New("max attempts must be greater than 0"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Publisher persists configured topics in the caller's transaction. Each successful
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
	cfg.OutboxTopics = slices.Clone(cfg.OutboxTopics)
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
	if !slices.Contains(p.cfg.OutboxTopics, topic) {
		return p.cfg.Publisher.Publish(topic, messages...)
	}
	for _, msg := range messages {
		if msg == nil {
			return errors.New("outbox message is nil")
		}
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
				SetTransactionID(tx.ID()).
				SetTopic(topic).
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
			p.cfg.Logger.WarnContext(p.ctx, "system event outbox drain encountered errors", "error", err)
		}
	}
}

func (p *Publisher) drain(ctx context.Context) error {
	var failedTransactions []string
	var errs []error
	progressed := false
	attempted := 0
	for attempted < p.cfg.DrainLimit {
		var transactionID string
		var publishErr error
		var exhausted []string
		sent := 0
		err := transaction.RunWithNoValue(ctx, enttx.NewCreator(p.cfg.DB), func(ctx context.Context) error {
			tx, err := entutils.GetDriverFromContext(ctx)
			if err != nil {
				return err
			}
			client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
			// Claim only a transaction's first pending row. Its lock prevents
			// other workers from claiming any sibling while this batch is sent.
			query := pendingTransactionQuery{Topics: p.cfg.OutboxTopics, MaxAttempts: p.cfg.MaxAttempts, ExcludedTransactions: failedTransactions}
			head, err := client.EventOutbox.Query().
				Where(query.Apply).
				Order(eventoutbox.ByID()).
				ForUpdate(sql.WithLockAction(sql.SkipLocked)).First(ctx)
			if db.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			transactionID = head.TransactionID
			// All siblings became visible in the same source commit. Do not use
			// SKIP LOCKED or a row limit here: the claim owns the whole group.
			rows, err := client.EventOutbox.Query().
				Where(
					eventoutbox.TopicIn(p.cfg.OutboxTopics...),
					eventoutbox.TransactionIDEQ(transactionID),
					eventoutbox.AttemptsLT(p.cfg.MaxAttempts),
				).
				Order(eventoutbox.ByID()).ForUpdate().All(ctx)
			if err != nil {
				return err
			}
			for _, row := range rows {
				attempted++
				msg := message.NewMessage(row.MessageID, row.Payload)
				msg.Metadata = message.Metadata(row.Metadata)
				msg.SetContext(ctx)
				if err := p.cfg.Publisher.Publish(row.Topic, msg); err != nil {
					publishErr = errors.Join(publishErr, fmt.Errorf("publish pending system event %s: %w", row.MessageID, err))
					if err := client.EventOutbox.UpdateOneID(row.ID).SetAttempts(row.Attempts + 1).Exec(ctx); err != nil {
						return fmt.Errorf("record failed delivery attempt: %w", err)
					}
					if row.Attempts+1 >= p.cfg.MaxAttempts {
						exhausted = append(exhausted, row.MessageID)
						continue
					}
					// Commit the successful prefix; this row and all later siblings
					// remain pending. Other source transactions can still progress.
					break
				}
				// Hard-delete acknowledged events: this table is a delivery queue.
				// Exhausted events stay stored with their failed-attempt count.
				if err := client.EventOutbox.DeleteOneID(row.ID).Exec(ctx); err != nil {
					return fmt.Errorf("remove delivered system event: %w", err)
				}
				sent++
			}
			return nil
		})
		if err == nil {
			if sent > 0 || len(exhausted) > 0 {
				progressed = true
			}
			for _, messageID := range exhausted {
				p.cfg.Logger.ErrorContext(ctx, "system event delivery abandoned after max attempts", "message_id", messageID, "transaction_id", transactionID, "attempts", p.cfg.MaxAttempts)
			}
		}
		if err := errors.Join(err, publishErr); err != nil {
			errs = append(errs, err)
			if transactionID == "" || ctx.Err() != nil {
				return errors.Join(errs...)
			}
			failedTransactions = append(failedTransactions, transactionID)
			continue
		}
		if transactionID == "" {
			return errors.Join(errs...)
		}
	}
	// Check the message budget between source transactions, never midway through
	// a successful batch. Continue productive work whose wakeups were coalesced.
	if progressed {
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
