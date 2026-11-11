package calculator

import (
	"context"
	"errors"
	"log/slog"

	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	customerbalance "github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Source interface {
	GetBalance(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error)
}

type Config struct {
	Source              Source
	Logger              *slog.Logger
	RatingReuseObserver func(context.Context, bool)
}

func (c Config) Validate() error {
	var errs []error

	if c.Source == nil {
		errs = append(errs, errors.New("source is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (rte.Calculator[creditbalance.Key, creditbalance.Balance], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	reuse, err := usagebasedrating.NewTotalsReuse(256)
	if err != nil {
		return nil, err
	}

	return &calculator{
		source:              config.Source,
		logger:              config.Logger,
		ratingReuse:         reuse,
		ratingReuseObserver: config.RatingReuseObserver,
	}, nil
}

type calculator struct {
	source              Source
	logger              *slog.Logger
	ratingReuse         *usagebasedrating.TotalsReuse
	ratingReuseObserver func(context.Context, bool)
}

func (c *calculator) Calculate(ctx context.Context, input rte.CalculateInput[creditbalance.Key]) (creditbalance.Balance, error) {
	if err := input.Validate(); err != nil {
		return creditbalance.Balance{}, err
	}

	ctx = usagebasedrating.WithTotalsReuse(ctx, c.ratingReuse, c.ratingReuseObserver)

	// Leave AsOf unset: historical queries intentionally omit live usage.
	balance, err := c.source.GetBalance(ctx, customerbalance.GetBalanceServiceInput{
		CustomerID:    input.Key.CustomerID,
		Currency:      input.Key.Currency,
		FeatureFilter: input.Key.FeatureFilter,
	})
	if err != nil {
		return creditbalance.Balance{}, err
	}

	return creditbalance.Balance{
		Booked:    balance.Booked(),
		Spendable: balance.Spendable(),
		Live:      balance.Live(),
		Pending:   balance.Pending(),
	}, nil
}
