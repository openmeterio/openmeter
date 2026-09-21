package service

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func (s *service) PlanIssue(ctx context.Context, input advance.IssueInput) ([]ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	originID := ulid.Make().String()

	inputs, err := transactions.ResolveTransactions(ctx, s.resolverDependencies(), transactions.ResolutionScope{
		CustomerID: input.CustomerID,
		Namespace:  input.CustomerID.Namespace,
	}, transactions.IssueCustomerReceivableTemplate{
		At:                 input.At,
		Amount:             input.Amount,
		Currency:           input.Currency,
		Features:           input.Features,
		SpendChargeID:      &input.ChargeID,
		CollectionOriginID: &originID,
	}, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
		At:                 input.At,
		Amount:             input.Amount,
		Currency:           input.Currency,
		Features:           input.Features,
		TaxCode:            input.TaxCode,
		TaxBehavior:        input.TaxBehavior,
		SpendChargeID:      &input.ChargeID,
		CollectionOriginID: &originID,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve advance transactions: %w", err)
	}

	return inputs, nil
}
