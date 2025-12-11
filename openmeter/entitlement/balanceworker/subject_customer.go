package balanceworker

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

// resolveCustomerAndSubject resolves the customer and optionally the subject.
// Subject may be nil if the customer has no usage attribution with subject keys.
func resolveCustomerAndSubject(ctx context.Context, customerService customer.Service, subjectService subject.Service, namespace string, customerID string) (customer.Customer, *subject.Subject, error) {
	cus, err := customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: namespace,
			ID:        customerID,
		},
	})
	if err != nil {
		return customer.Customer{}, nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Let's be defensive
	if cus == nil {
		return customer.Customer{}, nil, models.NewGenericNotFoundError(
			fmt.Errorf("customer not found [namespace=%s customer.id=%s]", namespace, customerID),
		)
	}

	// If no usage attribution, return customer without subject
	if cus.UsageAttribution == nil {
		return *cus, nil, nil
	}

	subjKey, err := cus.UsageAttribution.GetFirstSubjectKey()
	if err != nil {
		// No subject keys available - this is fine, just return customer without subject
		return *cus, nil, nil
	}

	subj, err := subjectService.GetByKey(ctx, models.NamespacedKey{
		Namespace: namespace,
		Key:       subjKey,
	})
	if err != nil {
		return customer.Customer{}, nil, fmt.Errorf("failed to get subject: %w", err)
	}

	return *cus, &subj, nil
}
