package correction

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (c *Corrector) Correct(ctx context.Context, input Input) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (creditrealization.CreateCorrectionInputs, error) {
		if len(input.Corrections) == 0 {
			return nil, nil
		}

		accounts, err := c.deps.AccountService.GetCustomerAccounts(ctx, customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		})
		if err != nil {
			return nil, err
		}

		if err := accounts.LockForPosting(ctx, c.deps.AccountCatalog); err != nil {
			return nil, err
		}

		plan, err := c.prepareCorrections(ctx, input)
		if err != nil {
			return nil, err
		}

		if len(plan.inputs) == 0 {
			return nil, nil
		}

		groupAnnotations := input.Annotations
		if groupAnnotations == nil {
			groupAnnotations = ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ChargeID,
			})
		}

		// Write the whole correction batch as one group and point every new correction
		// realization at that group.
		for i, txInput := range plan.inputs {
			if txInput != nil {
				plan.inputs[i] = transactions.WithAnnotations(txInput, groupAnnotations)
			}
		}

		transactionGroup, err := c.ledger.CommitGroup(ctx, transactions.GroupInputs(
			input.Namespace,
			groupAnnotations,
			plan.inputs...,
		))
		if err != nil {
			return nil, fmt.Errorf("commit correction transaction group: %w", err)
		}

		if err := c.breakage.PersistCommittedRecords(ctx, plan.breakagePending, transactionGroup); err != nil {
			return nil, fmt.Errorf("persist breakage records: %w", err)
		}

		for i := range plan.realizations {
			plan.realizations[i].LedgerTransaction = ledgertransaction.GroupReference{
				TransactionGroupID: transactionGroup.ID().ID,
			}
		}

		return plan.realizations, nil
	})
}
