package persistedstate

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	lines, err = slicesx.MapWithErr(lines, normalizePersistedLineOrHierarchy)
	if err != nil {
		return State{}, fmt.Errorf("normalizing existing lines: %w", err)
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

func normalizePersistedLineOrHierarchy(lineOrHierarchy billing.LineOrHierarchy) (billing.LineOrHierarchy, error) {
	// Subscription sync diffs against meter-compatible time windows. Historical persisted
	// lines can still carry sub-second timestamps from older writes, but the meter engine
	// only supports MinimumWindowSizeDuration precision. We normalize persisted state on
	// read so reconciliation does not keep proposing no-op repairs purely because the DB
	// preserved finer precision than the target state can legally represent.
	// TODO: Add a migration to normalize existing billing timestamps to the precision
	// supported by meter queries.
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := lineOrHierarchy.AsGenericLine()
		if err != nil {
			return billing.LineOrHierarchy{}, fmt.Errorf("getting line: %w", err)
		}

		cloned, err := line.Clone()
		if err != nil {
			return billing.LineOrHierarchy{}, fmt.Errorf("cloning line: %w", err)
		}

		cloned.UpdateServicePeriod(func(period *timeutil.ClosedPeriod) {
			*period = period.Truncate(streaming.MinimumWindowSizeDuration)
		})

		if invoiceAtAccessor, ok := cloned.(billing.InvoiceAtAccessor); ok {
			invoiceAtAccessor.SetInvoiceAt(invoiceAtAccessor.GetInvoiceAt().Truncate(streaming.MinimumWindowSizeDuration))
		}

		normalizeSubscriptionReference(cloned.GetSubscriptionReference())

		invoiceLine := cloned.AsInvoiceLine()
		switch invoiceLine.Type() {
		case billing.InvoiceLineTypeStandard:
			standardLine, err := invoiceLine.AsStandardLine()
			if err != nil {
				return billing.LineOrHierarchy{}, fmt.Errorf("getting standard line: %w", err)
			}

			return billing.NewLineOrHierarchy(&standardLine), nil
		case billing.InvoiceLineTypeGathering:
			gatheringLine, err := invoiceLine.AsGatheringLine()
			if err != nil {
				return billing.LineOrHierarchy{}, fmt.Errorf("getting gathering line: %w", err)
			}

			return billing.NewLineOrHierarchy(gatheringLine), nil
		default:
			return billing.LineOrHierarchy{}, fmt.Errorf("unsupported invoice line type: %s", invoiceLine.Type())
		}
	case billing.LineOrHierarchyTypeHierarchy:
		hierarchy, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return billing.LineOrHierarchy{}, fmt.Errorf("getting hierarchy: %w", err)
		}

		cloned, err := hierarchy.Clone()
		if err != nil {
			return billing.LineOrHierarchy{}, fmt.Errorf("cloning hierarchy: %w", err)
		}

		cloned.Group.ServicePeriod = cloned.Group.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)

		for i := range cloned.Lines {
			cloned.Lines[i].Line.UpdateServicePeriod(func(period *timeutil.ClosedPeriod) {
				*period = period.Truncate(streaming.MinimumWindowSizeDuration)
			})

			if invoiceAtAccessor, ok := cloned.Lines[i].Line.(billing.InvoiceAtAccessor); ok {
				invoiceAtAccessor.SetInvoiceAt(invoiceAtAccessor.GetInvoiceAt().Truncate(streaming.MinimumWindowSizeDuration))
			}

			normalizeSubscriptionReference(cloned.Lines[i].Line.GetSubscriptionReference())
		}

		return billing.NewLineOrHierarchy(&cloned), nil
	default:
		return lineOrHierarchy, nil
	}
}

func normalizeSubscriptionReference(ref *billing.SubscriptionReference) {
	if ref == nil {
		return
	}

	// Historical billing rows can carry sub-second subscription billing periods even
	// though subscription sync and meter queries operate on MinimumWindowSizeDuration
	// precision. Normalize the persisted subscription reference on read so legacy
	// timestamp precision does not leak into reconciliation decisions.
	// TODO: Add a migration to normalize existing billing timestamps to the precision
	// supported by meter queries.
	ref.BillingPeriod = ref.BillingPeriod.Truncate(streaming.MinimumWindowSizeDuration)
}
