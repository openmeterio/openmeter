package customer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(billingService billing.Service, syncService subscriptionsync.Service, subscriptionService subscription.Service) (*Validator, error) {
	if billingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if syncService == nil {
		return nil, fmt.Errorf("sync service is required")
	}

	return &Validator{
		billingService:      billingService,
		syncService:         syncService,
		subscriptionService: subscriptionService,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	billingService      billing.Service
	syncService         subscriptionsync.Service
	subscriptionService subscription.Service
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// A customer can only be deleted if all of his invocies are in final state

	if err := input.Validate(); err != nil {
		return err
	}

	// Let's sync any subscriptions pending for this customer
	subs, err := v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces:  []string{input.Namespace},
		CustomerIDs: []string{input.ID},
	})
	if err != nil {
		return err
	}

	watermark := time.Now().Add(-24 * time.Hour)

	for _, sub := range subs.Items {
		if sub.ActiveTo == nil || watermark.Before(*sub.ActiveTo) {
			view, err := v.subscriptionService.GetView(ctx, sub.NamespacedID)
			if err != nil {
				return err
			}

			if err := v.syncService.SynchronizeSubscription(ctx, view, time.Now()); err != nil {
				return err
			}
		}
	}

	invoices, err := v.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
	})
	if err != nil {
		return err
	}

	errs := make([]error, 0, len(invoices.Items))
	for _, inv := range invoices.Items {
		if inv.Type() == billing.InvoiceTypeGathering {
			gatheringInvoice, err := inv.AsGatheringInvoice()
			if err != nil {
				return err
			}

			if gatheringInvoice.DeletedAt != nil {
				continue
			}

			errs = append(errs, fmt.Errorf("invoice %s is still in gathering state", gatheringInvoice.ID))

			continue
		}

		stdInvoice, err := inv.AsStandardInvoice()
		if err != nil {
			return err
		}

		if !stdInvoice.Status.IsFinal() {
			errs = append(errs, fmt.Errorf("invoice %s is not in final state, please either delete the invoice or mark it uncollectible", stdInvoice.ID))
		}
	}

	return errors.Join(errs...)
}
