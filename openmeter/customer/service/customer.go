package customerservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

// ListCustomers lists customers
func (s *Service) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.PagedResponse[customer.Customer], error) {
		return s.adapter.ListCustomers(ctx, input)
	})
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
		// Delete the customer
		err := s.adapter.DeleteCustomer(ctx, input)
		if err != nil {
			return err
		}

		// Get the deleted customer
		deletedCustomer, err := s.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.ID,
			},
		})
		if err != nil {
			return err
		}

		// Publish the customer deleted event
		customerDeletedEvent := customer.NewCustomerDeleteEvent(ctx, deletedCustomer)
		if err := s.publisher.Publish(ctx, customerDeletedEvent); err != nil {
			return fmt.Errorf("failed to publish customer deleted event: %w", err)
		}

		return nil
	})
}

// GetCustomer gets a customer
func (s *Service) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		return s.adapter.GetCustomer(ctx, input)
	})
}

// GetCustomerByUsageAttribution gets a customer by usage attribution
func (s *Service) GetCustomerByUsageAttribution(ctx context.Context, input customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		return s.adapter.GetCustomerByUsageAttribution(ctx, input)
	})
}

// UpdateCustomer updates a customer
func (s *Service) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	// Validate the input
	if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		// Update the customer
		updatedCustomer, err := s.adapter.UpdateCustomer(ctx, input)
		if err != nil {
			return nil, err
		}

		// Publish the customer updated event
		customerUpdatedEvent := customer.NewCustomerUpdateEvent(ctx, updatedCustomer)
		if err := s.publisher.Publish(ctx, customerUpdatedEvent); err != nil {
			return nil, fmt.Errorf("failed to publish customer updated event: %w", err)
		}

		return updatedCustomer, nil
	})
}

// CustomerExists checks if a customer exists
func (s *Service) CustomerExists(ctx context.Context, customer customer.CustomerID) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.CustomerExists(ctx, customer)
	})
}
