package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// creditPurchaseHandler maps credit purchase lifecycle events to ledger transaction templates.
type creditPurchaseHandler struct {
	ledger          ledger.Ledger
	balanceQuerier  ledger.BalanceQuerier
	accountResolver ledger.AccountResolver
	accountCatalog  ledger.AccountCatalog

	advance            advance.Service
	breakage           breakage.Service
	transactionManager transaction.Creator
}

var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)

func NewCreditPurchaseHandler(
	ledger ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountCatalog ledger.AccountCatalog,
	advanceService advance.Service,
	breakageService breakage.Service,
	transactionManager transaction.Creator,
) (chargecreditpurchase.Handler, error) {
	if advanceService == nil {
		return nil, fmt.Errorf("advance service is required")
	}

	if breakageService == nil {
		breakageService = breakage.NewNoopService()
	}

	if transactionManager == nil {
		return nil, fmt.Errorf("transaction manager is required")
	}

	return &creditPurchaseHandler{
		ledger:             ledger,
		balanceQuerier:     balanceQuerier,
		accountResolver:    accountResolver,
		accountCatalog:     accountCatalog,
		advance:            advanceService,
		breakage:           breakageService,
		transactionManager: transactionManager,
	}, nil
}

func (h *creditPurchaseHandler) OnPromotionalCreditPurchase(ctx context.Context, input chargecreditpurchase.CreditGrantInput) (chargecreditpurchase.CreditGrantResult, error) {
	return h.issueCreditPurchase(ctx, input)
}

func (h *creditPurchaseHandler) OnCreditPurchaseInitiated(ctx context.Context, input chargecreditpurchase.CreditGrantInput) (chargecreditpurchase.CreditGrantResult, error) {
	return h.issueCreditPurchase(ctx, input)
}

func (h *creditPurchaseHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input chargecreditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	charge := input.Charge

	paymentPosting, err := resolveCreditPurchasePaymentPosting(input)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	customerID := charge.GetCustomerID()
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
	featureFilters := charge.Intent.FeatureFilters.Normalize()

	var templates []transactions.TransactionTemplate

	if charge.Intent.Currency.IsCustom() {
		templates = append(templates, transactions.ConvertCurrencyTemplate{
			At:             input.EventAt,
			SourceAmount:   paymentPosting.amount,
			TargetAmount:   charge.Intent.CreditAmount,
			CostBasis:      paymentPosting.costBasis,
			SourceCurrency: paymentPosting.currency,
			TargetCurrency: charge.Intent.Currency.Reference(),
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
		})
	}
	templates = append(templates, transactions.AuthorizeCustomerReceivablePaymentTemplate{
		At:             input.EventAt,
		Amount:         paymentPosting.amount,
		Currency:       paymentPosting.currency,
		CostBasis:      &paymentPosting.costBasis,
		Features:       featureFilters,
		SourceChargeID: &charge.ID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		templates...,
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	transactionGroup, err := h.commitTransactions(ctx, transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (h *creditPurchaseHandler) OnCreditPurchasePaymentSettled(ctx context.Context, input chargecreditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	charge := input.Charge

	paymentPosting, err := resolveCreditPurchasePaymentPosting(input)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	customerID := charge.GetCustomerID()
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
	featureFilters := charge.Intent.FeatureFilters.Normalize()

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:             input.EventAt,
			Amount:         paymentPosting.amount,
			Currency:       paymentPosting.currency,
			CostBasis:      &paymentPosting.costBasis,
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	transactionGroup, err := h.commitTransactions(ctx, transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

// issueCreditPurchase is the shared logic for issuing credits (both promotional and externally settled).
// It attributes outstanding advance receivables and unattributed accrued balances to the given cost basis,
// then issues new receivables for any remaining amount.
func (h *creditPurchaseHandler) issueCreditPurchase(ctx context.Context, input chargecreditpurchase.CreditGrantInput) (chargecreditpurchase.CreditGrantResult, error) {
	return transaction.Run(ctx, h.transactionManager, func(ctx context.Context) (chargecreditpurchase.CreditGrantResult, error) {
		charge := input.Charge

		if err := input.Validate(); err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		if charge.Intent.CreditAmount.IsZero() {
			return chargecreditpurchase.CreditGrantResult{}, nil
		}

		costBasis, err := input.GetCostBasis()
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		costBasisCurrency, err := input.GetCostBasisCurrency()
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		customerID := charge.GetCustomerID()

		accounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		if err := accounts.LockForPosting(ctx, h.accountCatalog); err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
		featureFilters := charge.Intent.FeatureFilters.Normalize()
		effectiveAt := charge.Intent.ServicePeriod.To

		// LedgerTransaction.CreatedAt retains recording time. For effective time,
		// future-effective purchases book attribution immediately so subsequent
		// committed purchases observe reduced advance; already-effective purchases
		// backdate attribution alongside issuance and settlement.
		advanceAttributionEffectiveAt := clock.Now()
		if effectiveAt.Before(advanceAttributionEffectiveAt) {
			advanceAttributionEffectiveAt = effectiveAt
		}

		// Advance attribution re-buckets an existing unknown-cost-basis advance into
		// this purchase's known cost-basis bucket, which requires a cost basis to
		// attribute into. A grant with no cost basis (a custom-currency promotional
		// grant) has none, so it skips attribution and issues its full amount to
		// FBO. The outstanding advance is left on its unknown-cost-basis route,
		// where the balance formula (FBO + nil-cost-basis advance receivable) still
		// nets it against the new credit, and a later paid purchase attributes it.
		var backfillPlan advance.BackfillPlan

		if costBasis != nil {
			backfillPlan, err = h.advance.PlanBackfill(ctx, advance.BackfillInput{
				CustomerID:        customerID,
				Currency:          charge.Intent.Currency,
				Amount:            charge.Intent.CreditAmount,
				At:                advanceAttributionEffectiveAt,
				CostBasis:         *costBasis,
				CostBasisCurrency: costBasisCurrency,
				Features:          featureFilters,
				SourceChargeID:    charge.ID,
				LegacyLineages:    input.AdvanceLineages,
			})
			if err != nil {
				return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("plan advance backfill: %w", err)
			}
		}

		for _, backfill := range backfillPlan.Backfills {
			if backfill.CollectionOriginID != nil {
				annotations[ledger.AnnotationBackfillCreditPriority] = lo.FromPtrOr(charge.Intent.Priority, ledger.DefaultCustomerFBOPriority)
			}
		}

		issuance := creditPurchaseIssuance{
			charge:            charge,
			costBasis:         costBasis,
			costBasisCurrency: costBasisCurrency,
			backfillPlan:      backfillPlan,
		}

		templates, err := issuance.buildTemplates()
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		inputs, err := transactions.ResolveTransactions(
			ctx,
			h.resolverDependencies(),
			transactions.ResolutionScope{
				CustomerID: customerID,
				Namespace:  charge.Namespace,
			},
			templates...,
		)
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("resolve transactions: %w", err)
		}

		var pendingBreakage []breakage.PendingRecord

		if breakageInput := issuance.mapBreakageInput(); breakageInput != nil {
			breakageInputs, pending, err := h.breakage.PlanIssuance(ctx, *breakageInput)
			if err != nil {
				return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("resolve breakage plan: %w", err)
			}

			inputs = append(inputs, breakageInputs...)
			pendingBreakage = pending
		}

		if len(inputs) == 0 {
			return chargecreditpurchase.CreditGrantResult{}, nil
		}

		transactionGroup, err := h.commitTransactions(ctx, transactions.GroupInputs(
			charge.Namespace,
			annotations,
			inputs...,
		))
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, err
		}

		if err := h.breakage.PersistCommittedRecords(ctx, pendingBreakage, transactionGroup); err != nil {
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("persist breakage records: %w", err)
		}

		return chargecreditpurchase.CreditGrantResult{
			GroupReference:      ledgertransaction.GroupReference{TransactionGroupID: transactionGroup.ID().ID},
			BackfillAllocations: backfillPlan.LegacyAllocations,
		}, nil
	})
}

// commitTransactions carries charge annotations on both the group and its transactions.
func (h *creditPurchaseHandler) commitTransactions(ctx context.Context, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	inputs := group.Transactions()

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, group.Annotations())
		}
	}

	committed, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(group.Namespace(), group.Annotations(), inputs...))
	if err != nil {
		return nil, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return committed, nil
}

func (h *creditPurchaseHandler) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService: h.accountResolver,
		AccountCatalog: h.accountCatalog,
		BalanceQuerier: h.balanceQuerier,
	}
}

type creditPurchasePaymentPosting struct {
	costBasis alpacadecimal.Decimal
	amount    alpacadecimal.Decimal
	currency  currencies.CurrencyReference
}

// resolveCreditPurchasePaymentPosting keeps fiat credit purchases in nominal
// credit units because their cost basis already carries the amount paid per
// unit. Custom-currency purchases instead settle the fiat receivable created by
// their FX transaction.
func resolveCreditPurchasePaymentPosting(input chargecreditpurchase.PaymentEventInput) (creditPurchasePaymentPosting, error) {
	charge := input.Charge

	if charge.State.ResolvedCostBasis == nil {
		return creditPurchasePaymentPosting{}, models.NewGenericPreConditionFailedError(
			fmt.Errorf("credit purchase charge[%s] cost basis is unresolved", charge.ID),
		)
	}

	if !charge.Intent.Currency.IsCustom() {
		return creditPurchasePaymentPosting{
			costBasis: charge.State.ResolvedCostBasis.CostBasis,
			amount:    charge.Intent.CreditAmount,
			currency:  charge.Intent.Currency.Reference(),
		}, nil
	}

	fiatCurrency, err := charge.Intent.GetSettlementFiatCurrency()
	if err != nil {
		return creditPurchasePaymentPosting{}, fmt.Errorf("get settlement fiat currency: %w", err)
	}

	return creditPurchasePaymentPosting{
		costBasis: charge.State.ResolvedCostBasis.CostBasis,
		amount:    input.FiatAmount,
		currency:  currencies.NewCurrencyReference(currencyx.Code(fiatCurrency.GetFiatCode())),
	}, nil
}
