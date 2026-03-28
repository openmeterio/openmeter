package persistedstate

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
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

func (l Loader) LoadForSubscription(ctx context.Context, subs subscription.Subscription) (State, error) {
	lines, err := l.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
		Namespace:      subs.Namespace,
		SubscriptionID: subs.ID,
		CustomerID:     subs.CustomerId,
	})
	if err != nil {
		return State{}, fmt.Errorf("getting existing lines: %w", err)
	}

	lines, err = slicesx.MapWithErr(lines, normalizePersistedLineOrHierarchy)
	if err != nil {
		return State{}, fmt.Errorf("normalizing existing lines: %w", err)
	}

	byUniqueID := make(map[string]Item, len(lines))
	for _, line := range lines {
		uniqueID := line.ChildUniqueReferenceID()
		if uniqueID == nil {
			continue
		}

		item, err := NewItemFromLineOrHierarchy(line)
		if err != nil {
			return State{}, fmt.Errorf("creating persisted item[%s]: %w", *uniqueID, err)
		}

		if _, ok := byUniqueID[*uniqueID]; ok {
			return State{}, fmt.Errorf("duplicate unique ids in the existing lines")
		}

		byUniqueID[*uniqueID] = item
	}

	invoices, err := l.loadInvoicesForSubscriptionLines(ctx, subs, lines)
	if err != nil {
		return State{}, err
	}

	return State{
		ByUniqueID: byUniqueID,
		Invoices:   invoices,
	}, nil
}

func (l Loader) loadInvoicesForSubscriptionLines(ctx context.Context, subs subscription.Subscription, lines []billing.LineOrHierarchy) (Invoices, error) {
	invoiceIDs := make(map[string]struct{})

	for _, line := range lines {
		switch line.Type() {
		case billing.LineOrHierarchyTypeLine:
			genericLine, err := line.AsGenericLine()
			if err != nil {
				return Invoices{}, fmt.Errorf("getting line invoice id: %w", err)
			}

			invoiceIDs[genericLine.GetInvoiceID()] = struct{}{}
		case billing.LineOrHierarchyTypeHierarchy:
			hierarchy, err := line.AsHierarchy()
			if err != nil {
				return Invoices{}, fmt.Errorf("getting hierarchy invoice ids: %w", err)
			}

			for _, child := range hierarchy.Lines {
				invoiceIDs[child.Invoice.GetID()] = struct{}{}
			}
		}
	}

	if len(invoiceIDs) == 0 {
		return Invoices{}, nil
	}

	invoices, err := l.loadInvoices(ctx, subs.Namespace, lo.Keys(invoiceIDs))
	if err != nil {
		return Invoices{}, err
	}

	for invoiceID := range invoiceIDs {
		if _, ok := invoices[invoiceID]; !ok {
			return Invoices{}, fmt.Errorf("invoice not found for persisted subscription state: %s", invoiceID)
		}
	}

	return invoices, nil
}

func (l Loader) loadInvoices(ctx context.Context, namespace string, invoiceIDs []string) (Invoices, error) {
	invoices, err := l.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		IDs:        invoiceIDs,
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

	return Invoices(byID), nil
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
