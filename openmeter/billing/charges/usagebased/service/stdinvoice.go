package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func (s *service) PostLineAssignedToInvoice(ctx context.Context, charge usagebased.Charge, line billing.GatheringLine) (creditrealization.Realizations, error) {
	return nil, fmt.Errorf("usage based invoice lifecycle is not implemented: %w", meta.ErrUnsupported)
}

func (s *service) PostInvoiceIssued(ctx context.Context, charge usagebased.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return fmt.Errorf("usage based invoice lifecycle is not implemented: %w", meta.ErrUnsupported)
}

func (s *service) PostPaymentAuthorized(ctx context.Context, charge usagebased.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return fmt.Errorf("usage based invoice lifecycle is not implemented: %w", meta.ErrUnsupported)
}

func (s *service) PostPaymentSettled(ctx context.Context, charge usagebased.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return fmt.Errorf("usage based invoice lifecycle is not implemented: %w", meta.ErrUnsupported)
}
