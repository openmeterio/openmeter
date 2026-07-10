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
			Expands:    customer.Expands{customer.ExpandSubscriptions},
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				return nil
			}

			return fmt.Errorf("failed to get customer [namespace=%s customer.id=%s]: %w",
				input.Namespace, input.ID, err)
		}

		if cus != nil {
			if cus.IsDeleted() {
				return nil
			}

			if cus.ActiveSubscriptionIDs.IsAbsent() {
				return fmt.Errorf("customer subscriptions are not expanded")
			}

			if len(cus.ActiveSubscriptionIDs.OrEmpty()) > 0 {
				return models.NewGenericPreConditionFailedError(
					customer.NewErrDeletingCustomerWithActiveSubscriptions(cus.ActiveSubscriptionIDs.OrEmpty()),
				)
			}
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

// GetCustomersByUsageAttribution resolves multiple customers by usage attribution keys in a single
// query, mapping each input key to the customer it matches with key-over-subject precedence applied.
func (s *Service) GetCustomersByUsageAttribution(ctx context.Context, input customer.GetCustomersByUsageAttributionInput) (map[string]customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error getting customers by usage attribution: %w", err),
		)
	}

	customers, err := s.adapter.GetCustomersByUsageAttribution(ctx, input)
	if err != nil {
		return nil, err
	}

	resolved, ambiguous := resolveCustomersByKey(customers, input.Keys)

	// A key resolving to both a distinct key-owner and a distinct subject-owner is not an error —
	// the key-owner wins deterministically, mirroring GetCustomerByUsageAttribution's single-row
	// precedence — but the underlying data (one key doing double duty as another customer's own
	// key and a different customer's subject key) is unusual enough to warrant operator attention,
	// so each occurrence is logged with the key and both customer IDs for follow-up.
	for _, m := range ambiguous {
		s.logger.ErrorContext(ctx, "ambiguous usage attribution key: matches both a customer key and a distinct subject key",
			"namespace", input.Namespace,
			"key", m.Key,
			"key_owner.id", m.KeyOwnerID,
			"subject_owner.id", m.SubjectOwnerID,
		)
	}

	return resolved, nil
}

// ambiguousUsageAttributionMatch records a key that matched both a distinct customer-key owner and
// a distinct subject-key owner, so the caller can log the identifiers needed to investigate and fix
// the underlying data overlap.
type ambiguousUsageAttributionMatch struct {
	Key            string
	KeyOwnerID     string
	SubjectOwnerID string
}

// resolveCustomersByKey maps each input key to the customer it matches. A key matches a customer
// either by the customer's own key or by one of its subject keys; when a key matches both a
// distinct key-owner and a distinct subject-owner, the key-owner takes precedence, mirroring the
// single-key GetCustomerByUsageAttribution's UNION ALL lookup_priority ordering. Keys with no match
// are absent from the returned map. The second return value lists the keys where precedence
// actually mattered (a distinct key-owner and a distinct subject-owner both matched).
func resolveCustomersByKey(customers []customer.Customer, keys []string) (map[string]customer.Customer, []ambiguousUsageAttributionMatch) {
	byKey := make(map[string]customer.Customer, len(customers))
	bySubject := make(map[string]customer.Customer, len(customers))

	for _, c := range customers {
		if c.Key != nil {
			byKey[*c.Key] = c
		}

		if c.UsageAttribution != nil {
			for _, sk := range c.UsageAttribution.SubjectKeys {
				if _, ok := bySubject[sk]; !ok {
					bySubject[sk] = c
				}
			}
		}
	}

	resolved := make(map[string]customer.Customer, len(keys))
	var ambiguous []ambiguousUsageAttributionMatch

	for _, k := range keys {
		keyOwner, hasKeyOwner := byKey[k]
		subjectOwner, hasSubjectOwner := bySubject[k]

		if hasKeyOwner && hasSubjectOwner && keyOwner.ID != subjectOwner.ID {
			ambiguous = append(ambiguous, ambiguousUsageAttributionMatch{
				Key:            k,
				KeyOwnerID:     keyOwner.ID,
				SubjectOwnerID: subjectOwner.ID,
			})
		}

		if hasKeyOwner {
			resolved[k] = keyOwner
			continue
		}

		if hasSubjectOwner {
			resolved[k] = subjectOwner
		}
	}

	return resolved, ambiguous
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
