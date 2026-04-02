package chargesworkeradvance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const defaultPageSize = 10_000

type AutoAdvancer struct {
	chargesService charges.ChargeService

	logger *slog.Logger
}

// All runs auto-advance for all eligible charges across all customers
func (a *AutoAdvancer) All(ctx context.Context, namespaces []string) error {
	a.logger.InfoContext(ctx, "listing customers with charges to advance")

	customers, err := a.ListCustomersToAdvance(ctx, namespaces)
	if err != nil {
		return fmt.Errorf("failed to list customers to advance charges: %w", err)
	}

	a.logger.DebugContext(ctx, "found customers with charges to advance", "count", len(customers))

	var errs []error
	for _, cust := range customers {
		if err := a.AdvanceCharges(ctx, cust); err != nil {
			a.logger.ErrorContext(ctx, "failed to auto-advance charges",
				slog.String("namespace", cust.Namespace),
				slog.String("customer_id", cust.ID),
				slog.String("error", err.Error()),
			)
			errs = append(errs, fmt.Errorf("failed to auto-advance charges [namespace=%s customer=%s]: %w", cust.Namespace, cust.ID, err))
		}
	}

	return errors.Join(errs...)
}

// ListCustomersToAdvance lists customers that have charges ready to be advanced
func (a *AutoAdvancer) ListCustomersToAdvance(ctx context.Context, namespaces []string) ([]customer.CustomerID, error) {
	now := time.Now()

	return pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) {
		return a.chargesService.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
			Page:            page,
			Namespaces:      namespaces,
			AdvanceAfterLTE: now,
		})
	}), defaultPageSize)
}

// AdvanceCharges advances all eligible charges for a customer
func (a *AutoAdvancer) AdvanceCharges(ctx context.Context, customerID customer.CustomerID) error {
	_, err := a.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	if err != nil {
		a.logger.WarnContext(ctx, "failed to advance charges",
			slog.String("namespace", customerID.Namespace),
			slog.String("customer_id", customerID.ID),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

type Config struct {
	ChargesService charges.ChargeService
	Logger         *slog.Logger
}

func NewAdvancer(config Config) (*AutoAdvancer, error) {
	if config.ChargesService == nil {
		return nil, fmt.Errorf("charges service is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &AutoAdvancer{
		chargesService: config.ChargesService,
		logger:         config.Logger,
	}, nil
}
