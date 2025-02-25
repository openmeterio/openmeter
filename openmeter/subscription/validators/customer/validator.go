package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(subscriptionService subscription.Service) (*Validator, error) {
	if subscriptionService == nil {
		return nil, fmt.Errorf("subscription service is required")
	}

	return &Validator{
		subscriptionService: subscriptionService,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	subscriptionService subscription.Service
}

// TODO: Wire in

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// A customer can only be deleted if all of his invocies are in final state

	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	subscriptions, err := v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{input.Namespace},
		Customers:  []string{input.ID},
		ActiveAt:   lo.ToPtr(clock.Now()),
	})
	if err != nil {
		return err
	}

	if len(subscriptions.Items) > 0 {
		return customer.ValidationError{
			Err: errors.New("customer has active subscriptions"),
		}
	}

	return nil
}
