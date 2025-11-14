package balanceworker

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

func resolveCustomerAndSubject(ctx context.Context, customerService customer.Service, subjectService subject.Service, namespace string, customerID string) (customer.Customer, subject.Subject, error) {
	cus, err := customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: namespace,
			ID:        customerID,
		},
	})
	if err != nil {
		return customer.Customer{}, subject.Subject{}, fmt.Errorf("failed to get customer: %w", err)
	}

	// Let's be defensive
	if cus == nil {
		return customer.Customer{}, subject.Subject{}, models.NewGenericNotFoundError(
			fmt.Errorf("customer not found [namespace=%s customer.id=%s]", namespace, customerID),
		)
	}

	subjKey, err := cus.UsageAttribution.GetFirstSubjectKey()
	if err != nil {
		return customer.Customer{}, subject.Subject{}, fmt.Errorf("failed to get subject key for customer %s: %w", customerID, err)
	}

	subj, err := subjectService.GetByKey(ctx, models.NamespacedKey{
		Namespace: namespace,
		Key:       subjKey,
	})
	if err != nil {
		return customer.Customer{}, subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
	}

	return *cus, subj, nil
}
