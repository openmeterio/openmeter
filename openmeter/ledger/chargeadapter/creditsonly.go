package chargeadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type creditsOnlyAccrualRequest struct {
	Namespace      string
	ChargeID       string
	CustomerID     string
	At             time.Time
	Currency       currencyx.Code
	SettlementMode productcatalog.SettlementMode
}

type CreditsOnlyUsageAccruedCorrectionInput struct {
	Namespace   string
	ChargeID    string
	AllocateAt  time.Time
	Corrections creditrealization.CorrectionRequest
}

type creditsOnlyCollectedSource struct {
	transaction                       ledger.Transaction
	group                             ledger.TransactionGroup
	advanceReceivableIssueTransaction ledger.Transaction
}

type creditsOnlyTransactionCorrectionPlan struct {
	transaction ledger.Transaction
	group       ledger.TransactionGroup
	amount      alpacadecimal.Decimal
}

func allocateCreditsToAccrued(
	ctx context.Context,
	ledgerService ledger.Ledger,
	deps transactions.ResolverDependencies,
	req creditsOnlyAccrualRequest,
	amount alpacadecimal.Decimal,
) (string, []ledger.TransactionInput, error) {
	customerID := customer.CustomerID{
		Namespace: req.Namespace,
		ID:        req.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: req.Namespace,
		ID:        req.ChargeID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  req.Namespace,
		},
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       req.At,
			Amount:   amount,
			Currency: req.Currency,
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("resolve transactions: %w", err)
	}

	collectedAmount := sumCollectedFBOAmount(inputs...)
	shortfall := amount.Sub(collectedAmount)
	if req.SettlementMode == productcatalog.CreditOnlySettlementMode && shortfall.IsPositive() {
		advanceInputs, err := transactions.ResolveTransactions(
			ctx,
			deps,
			transactions.ResolutionScope{
				CustomerID: customerID,
				Namespace:  req.Namespace,
			},
			transactions.IssueCustomerReceivableTemplate{
				At:       req.At,
				Amount:   shortfall,
				Currency: req.Currency,
			},
			transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
				At:       req.At,
				Amount:   shortfall,
				Currency: req.Currency,
			},
		)
		if err != nil {
			return "", nil, fmt.Errorf("resolve advance transactions: %w", err)
		}

		inputs = append(inputs, advanceInputs...)
	}

	if len(inputs) == 0 {
		return "", nil, nil
	}

	transactionGroup, err := ledgerService.CommitGroup(ctx, transactions.GroupInputs(
		req.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return "", nil, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return transactionGroup.ID().ID, inputs, nil
}

func correctCreditsOnlyAccrued(
	ctx context.Context,
	ledgerService ledger.Ledger,
	deps transactions.ResolverDependencies,
	input CreditsOnlyUsageAccruedCorrectionInput,
) (creditrealization.CreateCorrectionInputs, error) {
	if len(input.Corrections) == 0 {
		return nil, nil
	}

	plansByTransactionID, planOrder, err := planCreditsOnlyCorrections(ctx, ledgerService, input)
	if err != nil {
		return nil, err
	}

	resolvedInputs := make([]ledger.TransactionInput, 0, len(planOrder))
	for _, key := range planOrder {
		plan := plansByTransactionID[key]
		inputs, err := transactions.CorrectTransaction(ctx, deps, transactions.CorrectionInput{
			At:                  input.AllocateAt,
			Amount:              plan.amount,
			OriginalTransaction: plan.transaction,
			OriginalGroup:       plan.group,
		})
		if err != nil {
			return nil, fmt.Errorf("correct transaction %s: %w", plan.transaction.ID().ID, err)
		}
		resolvedInputs = append(resolvedInputs, inputs...)
	}

	if len(resolvedInputs) == 0 {
		return nil, nil
	}

	transactionGroup, err := ledgerService.CommitGroup(ctx, transactions.GroupInputs(
		input.Namespace,
		ledger.ChargeAnnotations(models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.ChargeID,
		}),
		resolvedInputs...,
	))
	if err != nil {
		return nil, fmt.Errorf("commit correction transaction group: %w", err)
	}

	out := make(creditrealization.CreateCorrectionInputs, 0, len(input.Corrections))
	for _, correction := range input.Corrections {
		out = append(out, creditrealization.CreateCorrectionInput{
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: transactionGroup.ID().ID,
			},
			Amount:                correction.Amount,
			CorrectsRealizationID: correction.Allocation.ID,
		})
	}

	return out, nil
}

func planCreditsOnlyCorrections(
	ctx context.Context,
	ledgerService ledger.Ledger,
	input CreditsOnlyUsageAccruedCorrectionInput,
) (map[string]*creditsOnlyTransactionCorrectionPlan, []string, error) {
	originalGroups, err := loadCreditsOnlyOriginalTransactionGroups(ctx, ledgerService, input.Namespace, input.Corrections)
	if err != nil {
		return nil, nil, err
	}

	plansByTransactionID := make(map[string]*creditsOnlyTransactionCorrectionPlan, len(input.Corrections))
	planOrder := make([]string, 0, len(input.Corrections))

	for _, originalGroup := range originalGroups {
		for _, correction := range originalGroup.Corrections {
			source, err := originalGroup.CollectedSourceForRealization(correction)
			if err != nil {
				return nil, nil, err
			}

			addTransactionCorrectionPlan(plansByTransactionID, &planOrder, source.transaction, source.group, correction.Amount.Abs())

			if source.advanceReceivableIssueTransaction != nil {
				addTransactionCorrectionPlan(plansByTransactionID, &planOrder, source.advanceReceivableIssueTransaction, source.group, correction.Amount.Abs())
			}
		}
	}

	return plansByTransactionID, planOrder, nil
}

type creditsOnlyOriginalTransactionGroup struct {
	Group            ledger.TransactionGroup
	Corrections      creditrealization.CorrectionRequest
	CollectedSources []creditsOnlyCollectedSource
}

func loadCreditsOnlyOriginalTransactionGroups(
	ctx context.Context,
	ledgerService ledger.Ledger,
	namespace string,
	corrections creditrealization.CorrectionRequest,
) ([]creditsOnlyOriginalTransactionGroup, error) {
	correctionsByGroupID := make(map[string]creditrealization.CorrectionRequest, len(corrections))
	groupOrder := make([]string, 0, len(corrections))

	for _, correction := range corrections {
		groupID := correction.Allocation.LedgerTransaction.TransactionGroupID
		if _, ok := correctionsByGroupID[groupID]; !ok {
			groupOrder = append(groupOrder, groupID)
		}
		correctionsByGroupID[groupID] = append(correctionsByGroupID[groupID], correction)
	}

	out := make([]creditsOnlyOriginalTransactionGroup, 0, len(groupOrder))
	for _, groupID := range groupOrder {
		group, err := ledgerService.GetTransactionGroup(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        groupID,
		})
		if err != nil {
			return nil, fmt.Errorf("get original transaction group %s: %w", groupID, err)
		}

		collectedSources, err := creditsOnlyCollectedSourcesForGroup(group)
		if err != nil {
			return nil, fmt.Errorf("map correction sources for group %s: %w", groupID, err)
		}

		out = append(out, creditsOnlyOriginalTransactionGroup{
			Group:            group,
			Corrections:      correctionsByGroupID[groupID],
			CollectedSources: collectedSources,
		})
	}

	return out, nil
}

func (g creditsOnlyOriginalTransactionGroup) CollectedSourceForRealization(
	correction creditrealization.CorrectionRequestItem,
) (creditsOnlyCollectedSource, error) {
	groupID := g.Group.ID().ID
	if correction.Allocation.SortHint < 0 || correction.Allocation.SortHint >= len(g.CollectedSources) {
		return creditsOnlyCollectedSource{}, fmt.Errorf("allocation sort hint %d out of range for transaction group %s", correction.Allocation.SortHint, groupID)
	}

	return g.CollectedSources[correction.Allocation.SortHint], nil
}

func addTransactionCorrectionPlan(plans map[string]*creditsOnlyTransactionCorrectionPlan, order *[]string, transaction ledger.Transaction, group ledger.TransactionGroup, amount alpacadecimal.Decimal) {
	key := transaction.ID().Namespace + ":" + transaction.ID().ID
	plan, ok := plans[key]
	if !ok {
		plans[key] = &creditsOnlyTransactionCorrectionPlan{
			transaction: transaction,
			group:       group,
			amount:      amount,
		}
		*order = append(*order, key)
		return
	}

	plan.amount = plan.amount.Add(amount)
}

func creditsOnlyCollectedSourcesForGroup(group ledger.TransactionGroup) ([]creditsOnlyCollectedSource, error) {
	out := make([]creditsOnlyCollectedSource, 0)
	for _, transaction := range group.Transactions() {
		templateName, err := ledger.TransactionTemplateNameFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template name: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}
		if direction != ledger.TransactionDirectionForward {
			continue
		}

		var advanceReceivableIssueTransaction ledger.Transaction
		if templateName == transactions.TemplateName(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			advanceReceivableIssueTransaction, err = findForwardTransactionByTemplate(group, transactions.TemplateName(transactions.IssueCustomerReceivableTemplate{}))
			if err != nil {
				return nil, fmt.Errorf("find issue receivable companion in group %s: %w", group.ID().ID, err)
			}
		}

		for _, entry := range transaction.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
				out = append(out, creditsOnlyCollectedSource{
					transaction:                       transaction,
					group:                             group,
					advanceReceivableIssueTransaction: advanceReceivableIssueTransaction,
				})
			}
		}
	}

	return out, nil
}

func findForwardTransactionByTemplate(group ledger.TransactionGroup, templateName string) (ledger.Transaction, error) {
	for _, transaction := range group.Transactions() {
		currentTemplateName, err := ledger.TransactionTemplateNameFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template name: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}

		if currentTemplateName == templateName && direction == ledger.TransactionDirectionForward {
			return transaction, nil
		}
	}

	return nil, fmt.Errorf("transaction with template %s not found", templateName)
}

func creditRealizationsFromCollectedInputs(servicePeriod timeutil.ClosedPeriod, transactionGroupID string, inputs ...ledger.TransactionInput) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				out = append(out, creditrealization.CreateAllocationInput{
					ServicePeriod: servicePeriod,
					Amount:        entry.Amount().Abs(),
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: transactionGroupID,
					},
				})
			}
		}
	}

	return out
}

func sumCollectedFBOAmount(inputs ...ledger.TransactionInput) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				total = total.Add(entry.Amount().Abs())
			}
		}
	}

	return total
}

func settledBalanceForSubAccount(ctx context.Context, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := subAccount.GetBalance(ctx)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
