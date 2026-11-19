package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/rte"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error
	if c.Client == nil {
		errs = append(errs, errors.New("postgres client is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (rte.UpstreamResolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &resolver{db: config.Client, logger: config.Logger}, nil
}

type resolver struct {
	db     *entdb.Client
	logger *slog.Logger
}

func (r *resolver) Resolve(ctx context.Context, event marshaler.Event) ([]rte.DemandSelector, error) {
	if lo.IsNil(event) {
		return nil, models.NewGenericValidationError(errors.New("source event is required"))
	}

	if err := event.Validate(); err != nil {
		return nil, err
	}

	source, ok := sourceEvents[event.EventName()]
	if !ok {
		return nil, nil
	}

	return source.resolve(r, ctx, event)
}

func (r *resolver) forIngest(ctx context.Context, e ingestevents.EventBatchedIngest) ([]rte.DemandSelector, error) {
	if len(e.RawEvents) == 0 {
		return nil, nil
	}

	// A newly created meter may already count these events while the sink
	// is still reporting slugs from its previous meter-cache refresh.
	var eventTypes []string
	timestamps := make([]time.Time, 0, len(e.RawEvents))
	for _, raw := range e.RawEvents {
		if raw.Type != "" {
			eventTypes = append(eventTypes, raw.Type)
		}

		timestamps = append(timestamps, time.Unix(raw.Time, 0).UTC())
	}

	if len(e.MeterSlugs) == 0 && len(eventTypes) == 0 {
		return nil, nil
	}

	return r.query(ctx, ingestQuery{
		namespace:  e.Namespace.ID,
		subject:    e.SubjectKey,
		meterSlugs: lo.Uniq(e.MeterSlugs),
		eventTypes: lo.Uniq(eventTypes),
		timestamps: lo.Uniq(timestamps),
		now:        clock.Now().UTC(),
	})
}

func entitlementSelector(namespace, id string) []rte.DemandSelector {
	return []rte.DemandSelector{{Namespace: namespace, Kind: rte.CalculationKindMeteredEntitlement, OwnerIDs: []string{id}}}
}

func (r *resolver) forCustomer(ctx context.Context, namespace, id string) ([]rte.DemandSelector, error) {
	return r.query(ctx, customerQuery{namespace: namespace, customerID: id, now: clock.Now().UTC()})
}

type sourceQuery interface{ SQL() (string, []any) }

func (r *resolver) query(ctx context.Context, query sourceQuery) ([]rte.DemandSelector, error) {
	statement, args := query.SQL()
	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("resolve affected RTE demand: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.ErrorContext(ctx, "close RTE upstream rows", "error", err)
		}
	}()
	var selectors []rte.DemandSelector
	type scope struct {
		namespace string
		kind      rte.CalculationKind
	}
	indexes := map[scope]int{}
	for rows.Next() {
		var namespace string
		var target rte.CalculationTarget
		var key []byte
		if err := rows.Scan(&namespace, &target.Kind, &key); err != nil {
			return nil, fmt.Errorf("scan affected RTE demand: %w", err)
		}

		target.Key = json.RawMessage(key)
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("stored RTE demand key: %w", err)
		}

		group := scope{namespace: namespace, kind: target.Kind}
		index, ok := indexes[group]
		if !ok {
			index = len(selectors)
			indexes[group] = index
			selectors = append(selectors, rte.DemandSelector{Namespace: namespace, Kind: target.Kind})
		}

		selectors[index].Targets = append(selectors[index].Targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate affected RTE demand: %w", err)
	}

	return selectors, nil
}
