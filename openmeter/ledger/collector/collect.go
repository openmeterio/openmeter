package collector

import (
	"context"
	"fmt"
	"maps"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/fbo"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type accrualCollector struct {
	ledger             ledger.Ledger
	fbo                fbo.Service
	transactionManager transaction.Creator
}

type collectedInputs []ledger.TransactionInput

func (c *accrualCollector) collectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (creditrealization.CreateAllocationInputs, error) {
		if input.Amount.IsZero() {
			return nil, nil
		}

		annotations := input.Annotations
		if annotations == nil {
			annotations = ledger.ChargeAnnotations(models.NamespacedID{Namespace: input.Namespace, ID: input.ChargeID})
		}

		scope, err := c.fbo.NewScope(ctx, fbo.ScopeInput{CustomerID: customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID}})
		if err != nil {
			return nil, err
		}

		advancePolicy := fbo.AdvancePolicyNever
		if input.SettlementMode == productcatalog.CreditOnlySettlementMode {
			advancePolicy = fbo.AdvancePolicyOnShortfall
		}

		plan, err := c.fbo.PlanConsume(ctx, scope, fbo.ConsumeInput{
			ChargeID:          input.ChargeID,
			Annotations:       annotations,
			BookedAt:          input.BookedAt,
			SourceBalanceAsOf: input.SourceBalanceAsOf,
			Currency:          input.Currency,
			Filters:           input.Filters,
			Amount:            input.Amount,
			Destination:       fbo.ConsumeDestinationAccrued,
			AdvancePolicy:     advancePolicy,
			TaxCode:           input.TaxCode,
			TaxBehavior:       input.TaxBehavior,
		})
		if err != nil {
			return nil, err
		}

		if len(plan.Inputs) == 0 {
			return nil, nil
		}

		groupInput, err := scope.GroupInput(ctx, annotations)
		if err != nil {
			return nil, err
		}

		group, err := c.ledger.CommitGroup(ctx, groupInput)
		if err != nil {
			return nil, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		return collectedInputs(plan.Inputs).toCreditRealizations(input.ServicePeriod, group.ID().ID), nil
	})
}

func (c *accrualCollector) collectToReceivable(ctx context.Context, input CollectToReceivableInput) (creditrealization.CreateAllocationInputs, error) {
	return transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (creditrealization.CreateAllocationInputs, error) {
		if input.Amount.IsZero() {
			return nil, nil
		}

		annotations := input.Annotations
		if annotations == nil {
			annotations = ledger.ChargeAnnotations(models.NamespacedID{Namespace: input.Namespace, ID: input.ChargeID})
		}

		scope, err := c.fbo.NewScope(ctx, fbo.ScopeInput{CustomerID: customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID}})
		if err != nil {
			return nil, err
		}

		plan, err := c.fbo.PlanConsume(ctx, scope, fbo.ConsumeInput{
			ChargeID:          input.ChargeID,
			Annotations:       annotations,
			BookedAt:          input.BookedAt,
			SourceBalanceAsOf: input.SourceBalanceAsOf,
			Currency:          input.Currency,
			Filters:           input.Filters,
			Amount:            input.Amount,
			Destination:       fbo.ConsumeDestinationReceivable,
			AdvancePolicy:     fbo.AdvancePolicyNever,
		})
		if err != nil {
			return nil, err
		}

		if len(plan.Inputs) == 0 {
			return nil, nil
		}

		groupInput, err := scope.GroupInput(ctx, annotations)
		if err != nil {
			return nil, err
		}

		group, err := c.ledger.CommitGroup(ctx, groupInput)
		if err != nil {
			return nil, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		return collectedInputs(plan.Inputs).toCreditRealizations(input.ServicePeriod, group.ID().ID), nil
	})
}

func (i collectedInputs) toCreditRealizations(servicePeriod timeutil.ClosedPeriod, transactionGroupID string) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(i))
	for _, input := range i {
		if input == nil {
			continue
		}

		annotations := maps.Clone(input.Annotations())
		if annotations == nil {
			annotations = make(models.Annotations)
		}

		annotations[ledger.AnnotationOriginTracked] = true
		// Keep billing realization granularity at the FBO sub-account bucket.
		// Entry identity may split same-sub-account collection internally, but
		// that should not leak as separate credit realizations.
		amountsBySubAccountID := make(map[string]alpacadecimal.Decimal)
		subAccountOrder := make([]string, 0)
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				subAccountID := entry.PostingAddress().SubAccountID()
				if _, ok := amountsBySubAccountID[subAccountID]; !ok {
					subAccountOrder = append(subAccountOrder, subAccountID)
				}

				amountsBySubAccountID[subAccountID] = amountsBySubAccountID[subAccountID].Add(entry.Amount().Abs())
			}
		}

		for _, subAccountID := range subAccountOrder {
			amount := amountsBySubAccountID[subAccountID]
			if !amount.IsPositive() {
				continue
			}

			out = append(out, creditrealization.CreateAllocationInput{
				Annotations:   annotations,
				ServicePeriod: servicePeriod,
				Amount:        amount,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: transactionGroupID,
				},
			})
		}
	}

	return out
}
