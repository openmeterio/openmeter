package customer

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(subscriptionService subscription.Service, customerService customer.Service) (*Validator, error) {
	if subscriptionService == nil {
		return nil, fmt.Errorf("subscription service is required")
	}
	if customerService == nil {
		return nil, fmt.Errorf("customer service is required")
	}

	return &Validator{
		subscriptionService: subscriptionService,
		customerService:     customerService,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	subscriptionService subscription.Service
	customerService     customer.Service
}

func (v *Validator) ValidateUpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// The subject association can only be changed if the customer doesn't have a subscription
	subscriptions, err := v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{input.CustomerID.Namespace},
		Customers:  []string{input.CustomerID.ID},
		ActiveAt:   lo.ToPtr(clock.Now()),
	})
	if err != nil {
		return err
	}

	hasSub := mo.Fold(subscriptions, func(list pagination.PagedResponse[subscription.SubscriptionView]) bool {
		return len(list.Items) > 0
	}, func(list pagination.PagedResponse[subscription.Subscription]) bool {
		return len(list.Items) > 0
	})

	if input.CustomerMutate.UsageAttribution.SubjectKeys != nil {
		currentCustomer, err := v.customerService.GetCustomer(ctx, customer.GetCustomerInput{
			Namespace: input.CustomerID.Namespace,
			ID:        input.CustomerID.ID,
		})
		if err != nil {
			return err
		}

		// Let's check the two subjectKey arrays are the same
		if hasSub {
			if len(currentCustomer.UsageAttribution.SubjectKeys) != len(input.CustomerMutate.UsageAttribution.SubjectKeys) {
				return fmt.Errorf("cannot change subject keys for customer with active subscriptions")
			}

			for i, key := range currentCustomer.UsageAttribution.SubjectKeys {
				if key != input.CustomerMutate.UsageAttribution.SubjectKeys[i] {
					return fmt.Errorf("cannot change subject keys for customer with active subscriptions")
				}
			}
		}
	}

	return nil
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// A customer can only be deleted if all of his invocies are in final state

	if err := input.Validate(); err != nil {
		return err
	}

	subscriptions, err := v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
		ActiveAt:   lo.ToPtr(clock.Now()),
	})
	if err != nil {
		return err
	}

	hasSub := mo.Fold(subscriptions, func(list pagination.PagedResponse[subscription.SubscriptionView]) bool {
		return len(list.Items) > 0
	}, func(list pagination.PagedResponse[subscription.Subscription]) bool {
		return len(list.Items) > 0
	})

	if hasSub {
		return fmt.Errorf("customer %s still have active subscriptions, please cancel them before deleting the customer", input.ID)
	}

	return nil
}
