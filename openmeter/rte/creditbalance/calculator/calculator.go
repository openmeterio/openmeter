package calculator

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	customerbalance "github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Source interface {
	GetBalance(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error)
}

type Config struct {
	Source Source
	Logger *slog.Logger
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

	return &calculator{
		source: config.Source,
		logger: config.Logger,
	}, nil
}

type calculator struct {
	source Source
	logger *slog.Logger
}

func (c *calculator) Calculate(ctx context.Context, input rte.CalculateInput[creditbalance.Key]) (creditbalance.Balance, error) {
	if err := input.Validate(); err != nil {
		return creditbalance.Balance{}, err
	}

	// Leave AsOf unset: historical queries intentionally omit live usage.
	balance, err := c.source.GetBalance(ctx, customerbalance.GetBalanceServiceInput{
		CustomerID:    input.Key.CustomerID,
		Currency:      input.Key.Currency,
		FeatureFilter: input.Key.FeatureFilter,
	})
	if err != nil {
		return creditbalance.Balance{}, err
	}

	return mapBalance(balance), nil
}

func mapBalance(balance customerbalance.Balance) creditbalance.Balance {
	return creditbalance.Balance{
		Booked:    balance.Booked(),
		Spendable: balance.Spendable(),
		Live:      balance.Live(),
		Pending:   balance.Pending(),
	}
}

type journalCheckpoint struct {
	Version int                                `json:"version"`
	Journal *customerbalance.JournalCheckpoint `json:"journal"`
}

type incrementalSource interface {
	GetIncrementalBalance(context.Context, customerbalance.GetIncrementalBalanceInput) (customerbalance.GetIncrementalBalanceResult, error)
}

func (c *calculator) CalculateIncremental(ctx context.Context, input rte.CalculateInput[creditbalance.Key], candidates []rte.Checkpoint) (rte.IncrementalResult[creditbalance.Balance], error) {
	source, ok := c.source.(incrementalSource)
	if !ok {
		value, err := c.Calculate(ctx, input)
		return rte.IncrementalResult[creditbalance.Balance]{Value: value}, err
	}

	if err := input.Validate(); err != nil {
		return rte.IncrementalResult[creditbalance.Balance]{}, err
	}

	request := customerbalance.GetIncrementalBalanceInput{
		StartedAt: input.At,
		GetBalanceServiceInput: customerbalance.GetBalanceServiceInput{
			CustomerID:    input.Key.CustomerID,
			Currency:      input.Key.Currency,
			FeatureFilter: input.Key.FeatureFilter,
		},
	}
	for _, cp := range candidates {
		var stored journalCheckpoint
		if err := json.Unmarshal(cp.Data, &stored); err == nil && stored.Version == 1 && stored.Journal != nil && stored.Journal.AsOf.Equal(cp.At) {
			request.Checkpoints = append(request.Checkpoints, *stored.Journal)
		}
	}

	result, err := source.GetIncrementalBalance(ctx, request)
	if err != nil {
		return rte.IncrementalResult[creditbalance.Balance]{}, err
	}

	output := rte.IncrementalResult[creditbalance.Balance]{Value: mapBalance(result.Balance)}
	switch {
	case result.Reused && result.Checkpoint != nil:
		output.Use = rte.CheckpointAdvanced
	case result.Reused:
		output.Use = rte.CheckpointReused
	}

	if result.Checkpoint != nil {
		data, err := json.Marshal(journalCheckpoint{Version: 1, Journal: result.Checkpoint})
		if err == nil {
			output.Checkpoint = &rte.Checkpoint{At: result.Checkpoint.AsOf, Data: data}
		}
	}

	return output, nil
}
