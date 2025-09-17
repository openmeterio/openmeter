package customerservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

// ListCustomers lists customers
func (s *Service) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

// ListCustomerUsageAttributions lists customer usage attributions
func (s *Service) ListCustomerUsageAttributions(ctx context.Context, input customer.ListCustomerUsageAttributionsInput) (pagination.Result[streaming.CustomerUsageAttribution], error) {
	return s.adapter.ListCustomerUsageAttributions(ctx, input)
}

// CreateCustomer creates a customer
func (s *Service) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	// Validate the input
	if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		// Create the customer
		createdCustomer, err := s.adapter.CreateCustomer(ctx, input)
		if err != nil {
			return nil, err
		}

		if err = s.hooks.PostCreate(ctx, createdCustomer); err != nil {
			return nil, err
		}

		// Publish the customer created event
		customerCreatedEvent := customer.NewCustomerCreateEvent(ctx, createdCustomer)
		if err := s.publisher.Publish(ctx, customerCreatedEvent); err != nil {
			return nil, fmt.Errorf("failed to publish customer created event: %w", err)
		}

		return createdCustomer, nil
	})
}

// DeleteCustomer deletes a customer
func (s *Service) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	// Validate the input
	if err := s.requestValidatorRegistry.ValidateDeleteCustomer(ctx, input); err != nil {
		return models.NewGenericValidationError(err)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		cus, err := s.adapter.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &input,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				return nil
			}

			return fmt.Errorf("failed to get customer [namespace=%s customer.id=%s]: %w",
				input.Namespace, input.ID, err)
		}

		if cus != nil && cus.IsDeleted() {
			return nil
		}

		if len(cus.ActiveSubscriptionIDs) > 0 {
			return models.NewGenericPreConditionFailedError(
				customer.NewErrDeletingCustomerWithActiveSubscriptions(cus.ActiveSubscriptionIDs),
			)
		}

		// Run pre delete hooks
		if err = s.hooks.PreDelete(ctx, cus); err != nil {
			return err
		}

		// Delete the customer
		err = s.adapter.DeleteCustomer(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to delete customer [namespace=%s customer.id=%s]: %w",
				input.Namespace, input.ID, err)
		}

		// Get the deleted customer
		cus, err = s.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.ID,
			},
		})
		if err != nil {
			return err
		}

		// Run post delete hooks
		if err = s.hooks.PostDelete(ctx, cus); err != nil {
			return err
		}

		// Publish the customer deleted event
		customerDeletedEvent := customer.NewCustomerDeleteEvent(ctx, cus)
		if err := s.publisher.Publish(ctx, customerDeletedEvent); err != nil {
			return fmt.Errorf("failed to publish customer deleted event: %w", err)
		}

		return nil
	})
}

// GetCustomer gets a customer
func (s *Service) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return s.adapter.GetCustomer(ctx, input)
}

// GetCustomerByUsageAttribution gets a customer by usage attribution
func (s *Service) GetCustomerByUsageAttribution(ctx context.Context, input customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	return s.adapter.GetCustomerByUsageAttribution(ctx, input)
}

// UpdateCustomer updates a customer
func (s *Service) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	// Validate the input
	if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		cus, err := s.adapter.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &input.CustomerID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get customer [namespace=%s customer.id=%s]: %w",
				input.CustomerID.Namespace, input.CustomerID.ID, err)
		}

		if cus != nil && cus.IsDeleted() {
			return nil, models.NewGenericPreConditionFailedError(
				fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
			)
		}

		// Run pre update hooks
		if err = s.hooks.PreUpdate(ctx, cus); err != nil {
			return nil, err
		}

		// Update the customer
		cus, err = s.adapter.UpdateCustomer(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to update customer [namespace=%s customer.id=%s]: %w",
				input.CustomerID.Namespace, input.CustomerID.ID, err)
		}

		// Run post update hooks
		if err = s.hooks.PostUpdate(ctx, cus); err != nil {
			return nil, err
		}

		// Publish the customer updated event
		customerUpdatedEvent := customer.NewCustomerUpdateEvent(ctx, cus)
		if err := s.publisher.Publish(ctx, customerUpdatedEvent); err != nil {
			return nil, fmt.Errorf("failed to publish customer updated event: %w", err)
		}

		return cus, nil
	})
}
