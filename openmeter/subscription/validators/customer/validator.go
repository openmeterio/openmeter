package customer

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
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
		Namespaces:  []string{input.CustomerID.Namespace},
		CustomerIDs: []string{input.CustomerID.ID},
		ActiveAt:    lo.ToPtr(clock.Now()),
	})
	if err != nil {
		return err
	}

	hasSub := len(subscriptions.Items) > 0

	// If there's an update to the subject keys we need additional checks
	if input.CustomerMutate.UsageAttribution != nil && input.CustomerMutate.UsageAttribution.SubjectKeys != nil {
		currentCustomer, err := v.customerService.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: input.CustomerID.Namespace,
				ID:        input.CustomerID.ID,
			},
		})
		if err != nil {
			return err
		}

		if currentCustomer != nil && currentCustomer.IsDeleted() {
			return models.NewGenericPreConditionFailedError(
				fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", currentCustomer.Namespace, currentCustomer.ID),
			)
		}

		if currentCustomer == nil {
			return models.NewGenericNotFoundError(
				fmt.Errorf("customer [namespace=%s customer.id=%s]", input.CustomerID.Namespace, input.CustomerID.ID),
			)
		}

		// Let's check the two subjectKey arrays are the same
		if hasSub {
			var currentSubjectKeys []string
			if currentCustomer.UsageAttribution != nil {
				currentSubjectKeys = currentCustomer.UsageAttribution.SubjectKeys
			}

			if len(currentSubjectKeys) != len(input.CustomerMutate.UsageAttribution.SubjectKeys) {
				return fmt.Errorf("cannot change subject keys for customer with active subscriptions")
			}

			for i, key := range currentSubjectKeys {
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
		Namespaces:  []string{input.Namespace},
		CustomerIDs: []string{input.ID},
		ActiveAt:    lo.ToPtr(clock.Now()),
	})
	if err != nil {
		return err
	}

	if len(subscriptions.Items) > 0 {
		return fmt.Errorf("customer %s still have active subscriptions, please cancel them before deleting the customer", input.ID)
	}

	return nil
}
