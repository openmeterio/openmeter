package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
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

	breakage           breakage.Service
	transactionManager transaction.Creator
}

var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)

func NewCreditPurchaseHandler(
	ledger ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountCatalog ledger.AccountCatalog,
	breakageService breakage.Service,
	transactionManager transaction.Creator,
) (chargecreditpurchase.Handler, error) {
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

	if charge.State.ResolvedCostBasis == nil {
		return ledgertransaction.GroupReference{}, models.NewGenericPreConditionFailedError(
			fmt.Errorf("credit purchase charge[%s] cost basis is unresolved", charge.ID),
		)
	}

	paymentPosting, err := resolveCreditPurchasePaymentPosting(input)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	costBasis := charge.State.ResolvedCostBasis.CostBasis

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
	featureFilters := charge.Intent.FeatureFilters.Normalize()

	var templates []transactions.TransactionTemplate
	if charge.Intent.Currency.IsCustom() {
		templates = append(templates, transactions.ConvertCurrencyTemplate{
			At:             input.EventAt,
			SourceAmount:   paymentPosting.amount,
			TargetAmount:   charge.Intent.CreditAmount,
			CostBasis:      costBasis,
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
		CostBasis:      &costBasis,
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

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, annotations)
		}
	}

	transactionGroupInput := transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	)
	transactionGroup, err := h.ledger.CommitGroup(ctx, transactionGroupInput)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
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

	if charge.State.ResolvedCostBasis == nil {
		return ledgertransaction.GroupReference{}, models.NewGenericPreConditionFailedError(
			fmt.Errorf("credit purchase charge[%s] cost basis is unresolved", charge.ID),
		)
	}

	paymentPosting, err := resolveCreditPurchasePaymentPosting(input)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	costBasis := charge.State.ResolvedCostBasis.CostBasis

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
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
			CostBasis:      &costBasis,
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, annotations)
		}
	}

	transactionGroupInput := transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	)
	transactionGroup, err := h.ledger.CommitGroup(ctx, transactionGroupInput)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
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

		var costBasisPtr *alpacadecimal.Decimal
		var costBasisCurrency *currencyx.Code
		if charge.Intent.Settlement.Type() == chargecreditpurchase.SettlementTypePromotional {
			if !charge.Intent.Currency.IsCustom() {
				costBasisPtr = lo.ToPtr(alpacadecimal.Zero)
			}
		} else {
			if charge.State.ResolvedCostBasis == nil {
				return chargecreditpurchase.CreditGrantResult{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("credit purchase charge[%s] cost basis is unresolved", charge.ID),
				)
			}

			costBasisPtr = &charge.State.ResolvedCostBasis.CostBasis
			if charge.Intent.Currency.IsCustom() {
				fiatCurrency, err := charge.Intent.GetSettlementFiatCurrency()
				if err != nil {
					return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("get settlement fiat currency: %w", err)
				}

				costBasisCurrency = lo.ToPtr(currencyx.Code(fiatCurrency.GetFiatCode()))
			}
		}
		customerID := customer.CustomerID{
			Namespace: charge.Namespace,
			ID:        charge.Intent.CustomerID,
		}

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
		var backfill advance.BackfillResult
		if costBasisPtr != nil {
			backfill, err = advance.PlanBackfill(ctx, advance.BackfillDependencies{
				Ledger:          h.ledger,
				BalanceQuerier:  h.balanceQuerier,
				AccountResolver: h.accountResolver,
			}, advance.BackfillInput{
				CustomerID:        customerID,
				Currency:          charge.Intent.Currency,
				Amount:            charge.Intent.CreditAmount,
				At:                advanceAttributionEffectiveAt,
				CostBasis:         *costBasisPtr,
				CostBasisCurrency: costBasisCurrency,
				Features:          featureFilters,
				SourceChargeID:    charge.ID,
				LegacyLineages:    input.AdvanceLineages,
			})
			if err != nil {
				return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("plan advance backfill: %w", err)
			}
		}

		for _, attribution := range backfill.Attributions {
			if attribution.CollectionOriginID != nil {
				annotations[ledger.AnnotationBackfillCreditPriority] = lo.FromPtrOr(charge.Intent.Priority, ledger.DefaultCustomerFBOPriority)
			}
		}

		issuableAmount := charge.Intent.CreditAmount.Sub(backfill.Amount)
		if issuableAmount.IsNegative() {
			issuableAmount = alpacadecimal.Zero
		}

		templates := backfill.Templates

		if issuableAmount.IsPositive() {
			templates = append(templates, transactions.IssueCustomerReceivableTemplate{
				At:                effectiveAt,
				Amount:            issuableAmount,
				Currency:          charge.Intent.Currency.Reference(),
				CostBasisCurrency: costBasisCurrency,
				CostBasis:         costBasisPtr,
				Features:          featureFilters,
				SourceChargeID:    &charge.ID,
				CreditPriority:    charge.Intent.Priority,
			})
		}

		switch charge.Intent.Settlement.Type() {
		case chargecreditpurchase.SettlementTypePromotional:
			// Promotional grants settle immediately through wash so the credited FBO balance
			// does not leave an unsettled receivable behind.
			templates = append(templates,
				transactions.AuthorizeCustomerReceivablePaymentTemplate{
					At:             effectiveAt,
					Amount:         charge.Intent.CreditAmount,
					Currency:       charge.Intent.Currency.Reference(),
					CostBasis:      costBasisPtr,
					Features:       featureFilters,
					SourceChargeID: &charge.ID,
				},
				transactions.SettleCustomerReceivableFromPaymentTemplate{
					At:             effectiveAt,
					Amount:         charge.Intent.CreditAmount,
					Currency:       charge.Intent.Currency.Reference(),
					CostBasis:      costBasisPtr,
					Features:       featureFilters,
					SourceChargeID: &charge.ID,
				},
			)
		case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
			// Deferred settlement modes are handled by later lifecycle events.
		default:
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
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
		if charge.Intent.ExpiresAt != nil {
			immediateReleases := make([]breakage.PlanIssuanceImmediateRelease, 0, len(backfill.Attributions))
			for _, attribution := range backfill.Attributions {
				if !attribution.Amount.IsPositive() {
					continue
				}

				immediateReleases = append(immediateReleases, breakage.PlanIssuanceImmediateRelease{
					Amount:             attribution.Amount,
					SpendChargeID:      attribution.SpendChargeID,
					CollectionOriginID: attribution.CollectionOriginID,
				})
			}

			breakageInputs, pending, err := h.breakage.PlanIssuance(ctx, breakage.PlanIssuanceInput{
				CustomerID:        customerID,
				Amount:            charge.Intent.CreditAmount,
				ImmediateReleases: immediateReleases,
				Currency:          charge.Intent.Currency.Reference(),
				CostBasisCurrency: costBasisCurrency,
				CostBasis:         costBasisPtr,
				CreditPriority:    charge.Intent.Priority,
				Features:          featureFilters,
				ExpiresAt:         *charge.Intent.ExpiresAt,
				SourceChargeID:    &charge.ID,
			})
			if err != nil {
				return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("resolve breakage plan: %w", err)
			}

			inputs = append(inputs, breakageInputs...)
			pendingBreakage = append(pendingBreakage, pending...)
		}

		if len(inputs) == 0 {
			return chargecreditpurchase.CreditGrantResult{}, nil
		}

		for i, input := range inputs {
			if input != nil {
				inputs[i] = transactions.WithAnnotations(input, annotations)
			}
		}

		transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
			charge.Namespace,
			annotations,
			inputs...,
		))
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		if err := h.breakage.PersistCommittedRecords(ctx, pendingBreakage, transactionGroup); err != nil {
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("persist breakage records: %w", err)
		}

		return chargecreditpurchase.CreditGrantResult{
			GroupReference:      ledgertransaction.GroupReference{TransactionGroupID: transactionGroup.ID().ID},
			BackfillAllocations: backfill.LegacyAllocations,
		}, nil
	})
}

func (h *creditPurchaseHandler) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService: h.accountResolver,
		AccountCatalog: h.accountCatalog,
		BalanceQuerier: h.balanceQuerier,
	}
}

type creditPurchasePaymentPosting struct {
	amount   alpacadecimal.Decimal
	currency currencies.CurrencyReference
}

// resolveCreditPurchasePaymentPosting keeps fiat credit purchases in nominal
// credit units because their cost basis already carries the amount paid per
// unit. Custom-currency purchases instead settle the fiat receivable created by
// their FX transaction.
func resolveCreditPurchasePaymentPosting(input chargecreditpurchase.PaymentEventInput) (creditPurchasePaymentPosting, error) {
	charge := input.Charge
	if !charge.Intent.Currency.IsCustom() {
		return creditPurchasePaymentPosting{
			amount:   charge.Intent.CreditAmount,
			currency: charge.Intent.Currency.Reference(),
		}, nil
	}

	fiatCurrency, err := charge.Intent.GetSettlementFiatCurrency()
	if err != nil {
		return creditPurchasePaymentPosting{}, fmt.Errorf("get settlement fiat currency: %w", err)
	}

	return creditPurchasePaymentPosting{
		amount:   input.FiatAmount,
		currency: currencies.NewCurrencyReference(currencyx.Code(fiatCurrency.GetFiatCode())),
	}, nil
}
