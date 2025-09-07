package customer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(billingService billing.Service, entitlementService entitlement.Connector, syncService *billingworkersubscription.Handler, subscriptionService subscription.Service) (*Validator, error) {
	if billingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if syncService == nil {
		return nil, fmt.Errorf("sync service is required")
	}

	return &Validator{
		billingService:      billingService,
		entitlementService:  entitlementService,
		syncService:         syncService,
		subscriptionService: subscriptionService,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	billingService      billing.Service
	entitlementService  entitlement.Connector
	syncService         *billingworkersubscription.Handler
	subscriptionService subscription.Service
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// A customer can only be deleted if all of his invocies are in final state

	if err := input.Validate(); err != nil {
		return err
	}

	// Let's sync any subscriptions pending for this customer
	subs, err := v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
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

			if err := v.syncService.SyncronizeSubscription(ctx, view, time.Now()); err != nil {
				return err
			}
		}
	}

	gatheringInvoices, err := v.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
	})
	if err != nil {
		return err
	}

	errs := make([]error, 0, len(gatheringInvoices.Items))
	for _, inv := range gatheringInvoices.Items {
		if inv.Status == billing.InvoiceStatusGathering {
			errs = append(errs, fmt.Errorf("invoice %s is still in gathering state", inv.ID))

			continue
		}

		if !inv.Status.IsFinal() {
			errs = append(errs, fmt.Errorf("invoice %s is not in final state, please either delete the invoice or mark it uncollectible", inv.ID))
		}
	}

	// Check if the customer has any entitlements
	access, err := v.entitlementService.GetAccess(ctx, input.Namespace, input.ID)
	if err != nil {
		return err
	}
	if len(access.Entitlements) > 0 {
		errs = append(errs, fmt.Errorf("customer has entitlements, please remove them before deleting the customer"))
	}

	return errors.Join(errs...)
}
