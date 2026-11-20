// Package client provides access to RTE without constructing a calculation engine.
package client

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	DB        *db.Client
	Publisher eventbus.Publisher
	Logger    *slog.Logger
	Meter     metric.Meter
}

type client struct {
	credit  rte.BalanceClient[creditbalance.Key, creditbalance.Balance]
	metered rte.BalanceClient[meteredentitlement.Key, meteredentitlement.Balance]
}

func New(config Config) (rte.Client, error) {
	if config.Publisher == nil {
		return nil, errors.New("RTE request publisher is required")
	}

	if config.Meter == nil {
		return nil, errors.New("RTE client meter is required")
	}

	cfg := adapter.Config{Client: config.DB, Logger: config.Logger}
	credit, err := adapter.NewCreditBalanceStore(cfg)
	if err != nil {
		return nil, err
	}

	metered, err := adapter.NewMeteredEntitlementBalanceStore(cfg)
	if err != nil {
		return nil, err
	}

	demand, err := adapter.NewDemandRepository(cfg)
	if err != nil {
		return nil, err
	}

	creditReads, err := metrics.NewStateReads(config.Meter, rte.CalculationKindCreditBalance)
	if err != nil {
		return nil, err
	}

	meteredReads, err := metrics.NewStateReads(config.Meter, rte.CalculationKindMeteredEntitlement)
	if err != nil {
		return nil, err
	}

	return &client{
		credit:  balances[creditbalance.Key, creditbalance.Balance]{reader: credit, metrics: creditReads, demand: demand, publisher: config.Publisher},
		metered: balances[meteredentitlement.Key, meteredentitlement.Balance]{reader: metered, metrics: meteredReads, demand: demand, publisher: config.Publisher},
	}, nil
}

func (c *client) CreditBalances() rte.BalanceClient[creditbalance.Key, creditbalance.Balance] {
	return c.credit
}

func (c *client) MeteredEntitlements() rte.BalanceClient[meteredentitlement.Key, meteredentitlement.Balance] {
	return c.metered
}

type reader[K rte.Key, V any] interface {
	Get(context.Context, K) (rte.State[V], error)
}

type balances[K rte.Key, V any] struct {
	reader    reader[K, V]
	metrics   metrics.StateReads
	demand    rte.DemandRepository
	publisher eventbus.Publisher
}

func (b balances[K, V]) Get(ctx context.Context, key K) (rte.State[V], error) {
	if err := key.Validate(); err != nil {
		return rte.State[V]{}, err
	}

	start := time.Now()
	state, err := b.reader.Get(ctx, key)
	switch {
	case err == nil:
		b.metrics.Read(ctx, start, metrics.ReadOutcomeHit)
	case models.IsGenericNotFoundError(err):
		b.metrics.Read(ctx, start, metrics.ReadOutcomeMissing)
	default:
		b.metrics.Read(ctx, start, metrics.ReadOutcomeFailed)
	}

	return state, err
}

func (b balances[K, V]) RequestCalculation(ctx context.Context, key K) error {
	return b.request(ctx, key, nil)
}

func (b balances[K, V]) RequestRefresh(ctx context.Context, key K, freshness rte.FreshnessPolicy) error {
	return b.request(ctx, key, &freshness)
}

func (b balances[K, V]) request(ctx context.Context, key K, policy *rte.FreshnessPolicy) error {
	request, err := rte.NewCalculationRequestedEvent(key)
	if err != nil {
		return err
	}

	request.Freshness = policy
	if err := request.Validate(); err != nil {
		return err
	}

	return b.publisher.Publish(ctx, request)
}

func (b balances[K, V]) Activate(ctx context.Context, key K) error {
	target, err := rte.NewCalculationTarget(key)
	if err != nil {
		return err
	}

	return b.demand.Activate(ctx, target)
}

func (b balances[K, V]) Deactivate(ctx context.Context, key K) error {
	target, err := rte.NewCalculationTarget(key)
	if err != nil {
		return err
	}

	return b.demand.Deactivate(ctx, target)
}
