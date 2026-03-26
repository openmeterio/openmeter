package persistedstate

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type billingService interface {
	GetLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]billing.LineOrHierarchy, error)
	ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error)
}

type Loader struct {
	billingService billingService
}

func NewLoader(billingService billingService) Loader {
	return Loader{
		billingService: billingService,
	}
}

func (l Loader) LoadForSubscription(ctx context.Context, subs subscription.SubscriptionView) (State, error) {
	lines, err := l.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
		Namespace:      subs.Subscription.Namespace,
		SubscriptionID: subs.Subscription.ID,
		CustomerID:     subs.Subscription.CustomerId,
	})
	if err != nil {
		return State{}, fmt.Errorf("getting existing lines: %w", err)
	}

	byUniqueID, unique := slicesx.UniqueGroupBy(
		lo.Filter(lines, func(line billing.LineOrHierarchy, _ int) bool {
			return line.ChildUniqueReferenceID() != nil
		}),
		func(line billing.LineOrHierarchy) string {
			return *line.ChildUniqueReferenceID()
		},
	)
	if !unique {
		return State{}, fmt.Errorf("duplicate unique ids in the existing lines")
	}

	return State{
		Lines:      lines,
		ByUniqueID: byUniqueID,
	}, nil
}

func (l Loader) LoadInvoicesForCustomer(ctx context.Context, customerID customer.CustomerID) (Invoices, error) {
	invoices, err := l.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{customerID.Namespace},
		Customers:  []string{customerID.ID},
	})
	if err != nil {
		return Invoices{}, fmt.Errorf("listing invoices: %w", err)
	}

	byID := make(map[string]billing.Invoice, len(invoices.Items))
	for _, invoice := range invoices.Items {
		genericInvoice, err := invoice.AsGenericInvoice()
		if err != nil {
			return Invoices{}, fmt.Errorf("converting invoice to generic invoice: %w", err)
		}

		byID[genericInvoice.GetID()] = invoice
	}

	return Invoices{
		ByID: byID,
	}, nil
}
