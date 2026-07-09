package chargeadapter

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/cmpx"
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

type fundedSettlementLedgerTerms struct {
	CreditCurrency     currencyx.Code
	SettlementCurrency currencyx.Code
	CreditAmount       alpacadecimal.Decimal
	SettlementAmount   alpacadecimal.Decimal
	CostBasis          alpacadecimal.Decimal
	Source             *currencyx.Code
	NeedsReceivableFX  bool
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

func ledgerTermsForCreditPurchase(charge chargecreditpurchase.Charge) (fundedSettlementLedgerTerms, error) {
	terms := fundedSettlementLedgerTerms{
		CreditCurrency:     charge.Intent.Currency,
		SettlementCurrency: charge.Intent.Currency,
		CreditAmount:       charge.Intent.CreditAmount,
		SettlementAmount:   charge.Intent.CreditAmount,
	}

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return fundedSettlementLedgerTerms{}, fmt.Errorf("get cost basis: %w", err)
	}
	terms.CostBasis = costBasis

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		return terms, nil
	case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
		settlementCurrency, settlementAmount, err := chargecreditpurchase.SettlementAmount(charge.Intent.Settlement, charge.Intent.CreditAmount)
		if err != nil {
			return fundedSettlementLedgerTerms{}, fmt.Errorf("settlement amount: %w", err)
		}

		if charge.Intent.Currency.IsKnownFiat() {
			return terms, nil
		}

		terms.SettlementCurrency = settlementCurrency
		terms.SettlementAmount = settlementAmount
		terms.Source = &settlementCurrency
		terms.NeedsReceivableFX = true

		return terms, nil
	default:
		return fundedSettlementLedgerTerms{}, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
	}
}

func (h *creditPurchaseHandler) OnPromotionalCreditPurchase(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchaseInitiated(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input chargecreditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}
	charge := input.Charge

	terms, err := ledgerTermsForCreditPurchase(charge)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

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
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:             input.EventAt,
			Amount:         terms.SettlementAmount,
			Currency:       terms.SettlementCurrency,
			CostBasis:      &terms.CostBasis,
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

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	))
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

	terms, err := ledgerTermsForCreditPurchase(charge)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

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
			Amount:         terms.SettlementAmount,
			Currency:       terms.SettlementCurrency,
			CostBasis:      &terms.CostBasis,
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

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	))
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
func (h *creditPurchaseHandler) issueCreditPurchase(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return transaction.Run(ctx, h.transactionManager, func(ctx context.Context) (ledgertransaction.GroupReference, error) {
		return h.issueCreditPurchaseGroup(ctx, charge)
	})
}

func (h *creditPurchaseHandler) issueCreditPurchaseGroup(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Intent.CreditAmount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	terms, err := ledgerTermsForCreditPurchase(charge)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
	featureFilters := charge.Intent.FeatureFilters.Normalize()
	bookedAt := charge.Intent.ServicePeriod.To

	advanceAttributions, err := h.advanceAttributions(ctx, customerID, terms.CreditCurrency, terms.CreditAmount, featureFilters)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get advance attributions: %w", err)
	}

	advanceAttributionAmount := alpacadecimal.Zero
	for _, attribution := range advanceAttributions {
		advanceAttributionAmount = advanceAttributionAmount.Add(attribution.advanceAmount)
	}

	issuableAmount := terms.CreditAmount.Sub(advanceAttributionAmount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	var templates []transactions.TransactionTemplate

	for _, attribution := range advanceAttributions {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:                 bookedAt,
			Amount:             attribution.advanceAmount,
			Currency:           terms.CreditCurrency,
			CostBasis:          &terms.CostBasis,
			Source:             terms.Source,
			AdvanceFeatures:    attribution.advanceFeatures,
			AttributedFeatures: featureFilters,
			SourceChargeID:     &charge.ID,
			SpendChargeID:      attribution.spendChargeID,
		})

		if attribution.accruedAmount.IsPositive() {
			templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:             bookedAt,
				Amount:         attribution.accruedAmount,
				Currency:       terms.CreditCurrency,
				TaxCode:        attribution.taxCode,
				TaxBehavior:    attribution.taxBehavior,
				FromCostBasis:  nil,
				ToCostBasis:    &terms.CostBasis,
				Source:         terms.Source,
				SourceChargeID: &charge.ID,
				SpendChargeID:  attribution.spendChargeID,
			})
		}
	}

	if issuableAmount.IsPositive() {
		templates = append(templates, transactions.IssueCustomerReceivableTemplate{
			At:             bookedAt,
			Amount:         issuableAmount,
			Currency:       terms.CreditCurrency,
			CostBasis:      &terms.CostBasis,
			Source:         terms.Source,
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
			CreditPriority: charge.Intent.Priority,
		})
	}

	if terms.NeedsReceivableFX {
		templates = append(templates, transactions.ConvertCustomerReceivableCurrencyTemplate{
			At:             bookedAt,
			SourceAmount:   terms.CreditAmount,
			SourceCurrency: terms.CreditCurrency,
			TargetCurrency: terms.SettlementCurrency,
			CostBasis:      terms.CostBasis,
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
		})
	}

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		// Promotional grants settle immediately through wash so the credited FBO balance
		// does not leave an unsettled receivable behind.
		templates = append(templates,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:             bookedAt,
				Amount:         terms.SettlementAmount,
				Currency:       terms.SettlementCurrency,
				CostBasis:      &terms.CostBasis,
				Features:       featureFilters,
				SourceChargeID: &charge.ID,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:             bookedAt,
				Amount:         terms.SettlementAmount,
				Currency:       terms.SettlementCurrency,
				CostBasis:      &terms.CostBasis,
				Features:       featureFilters,
				SourceChargeID: &charge.ID,
			},
		)
	case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
		// Deferred settlement modes are handled by later lifecycle events.
	default:
		return ledgertransaction.GroupReference{}, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
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
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	var pendingBreakage []breakage.PendingRecord
	if charge.Intent.ExpiresAt != nil {
		immediateReleases := make([]breakage.PlanIssuanceImmediateRelease, 0, len(advanceAttributions))
		for _, attribution := range advanceAttributions {
			if !attribution.advanceAmount.IsPositive() {
				continue
			}

			immediateReleases = append(immediateReleases, breakage.PlanIssuanceImmediateRelease{
				Amount:        attribution.advanceAmount,
				SpendChargeID: attribution.spendChargeID,
			})
		}

		breakageInputs, pending, err := h.breakage.PlanIssuance(ctx, breakage.PlanIssuanceInput{
			CustomerID:        customerID,
			Amount:            terms.CreditAmount,
			ImmediateReleases: immediateReleases,
			Currency:          terms.CreditCurrency,
			Source:            terms.Source,
			CostBasis:         &terms.CostBasis,
			CreditPriority:    charge.Intent.Priority,
			Features:          featureFilters,
			ExpiresAt:         *charge.Intent.ExpiresAt,
			SourceChargeID:    &charge.ID,
		})
		if err != nil {
			return ledgertransaction.GroupReference{}, fmt.Errorf("resolve breakage plan: %w", err)
		}

		inputs = append(inputs, breakageInputs...)
		pendingBreakage = append(pendingBreakage, pending...)
	}

	if len(inputs) == 0 {
		return ledgertransaction.GroupReference{}, nil
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
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	if err := h.breakage.PersistCommittedRecords(ctx, pendingBreakage, transactionGroup); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("persist breakage records: %w", err)
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (h *creditPurchaseHandler) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService: h.accountResolver,
		AccountCatalog: h.accountCatalog,
		BalanceQuerier: h.balanceQuerier,
	}
}

// advanceAttribution is the posting plan for one slice of existing advance
// exposure that a credit purchase can backfill. It carries the original spend
// charge so receivable and accrued translations preserve downstream revenue
// provenance after source charge attribution.
type advanceAttribution struct {
	taxCode         *string
	taxBehavior     *ledger.TaxBehavior
	advanceFeatures []string
	spendChargeID   *string
	advanceAmount   alpacadecimal.Decimal
	accruedAmount   alpacadecimal.Decimal
}

// unattributedAccruedBalance is source-less accrued value available for
// creditpurchase backfill. It is keyed by the dimensions that must be preserved
// during cost-basis translation: tax treatment and spend charge provenance.
type unattributedAccruedBalance struct {
	key         accruedBackfillBucketKey
	taxCode     *string
	taxBehavior *ledger.TaxBehavior
	amount      alpacadecimal.Decimal
}

// taxDimensionKey keeps tax-bearing accrued balances separate because
// backfilling credit source/cost basis must not merge taxable and non-taxable
// accrued buckets.
type taxDimensionKey struct {
	taxCode     string
	taxBehavior string
}

// accruedBackfillBucketKey adds the accrued dimensions that must remain split
// during cost-basis translation after a receivable bucket has matched by spend.
type accruedBackfillBucketKey struct {
	spendChargeID string
	taxDimensionKey
}

// advanceReceivableBuckets is the mutable allocation state for source-less
// advance receivable. Matching happens by spend charge when it exists; legacy
// rows have no spend charge, so each route bucket remains separate inside the
// same spend group and is consumed in deterministic route order.
type advanceReceivableBuckets struct {
	bySpendChargeID        map[string][]advanceReceivableBalance
	remainingBySpendKey    map[string]alpacadecimal.Decimal
	orderedSpendChargeKeys []string
	total                  alpacadecimal.Decimal
}

// advanceAttributions determines how much of a credit purchase first covers
// existing advance receivable and accrued exposure before issuing new credit.
// It matches receivable and accrued buckets by spend charge so source attribution
// does not move value from one spending charge into another charge's provenance.
// Legacy rows have no spend charge; for those, route buckets still need to stay
// distinct so clearing receivable cannot accidentally net across feature routes.
func (h *creditPurchaseHandler) advanceAttributions(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	amount alpacadecimal.Decimal,
	creditFeatures []string,
) ([]advanceAttribution, error) {
	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	advanceReceivables, err := h.advanceReceivableBalances(ctx, customerAccounts.ReceivableAccount.ID(), currency)
	if err != nil {
		return nil, fmt.Errorf("list advance receivable balances: %w", err)
	}
	slices.SortStableFunc(advanceReceivables, cmpx.Compare[advanceReceivableBalance])

	unattributedAccrued, err := h.unattributedAccruedBalances(ctx, customerAccounts.AccruedAccount, currency)
	if err != nil {
		return nil, err
	}

	receivableBuckets := newAdvanceReceivableBuckets(advanceReceivables, creditFeatures)

	// The purchase can only attribute as much as both the purchase and matching
	// open advance receivable allow. Anything left after this becomes ordinary
	// issued credit later in the handler.
	attributed := receivableBuckets.total
	if attributed.GreaterThan(amount) {
		attributed = amount
	}

	if !attributed.IsPositive() {
		return nil, nil
	}

	// If matching source-less accrued exists, translate accrued into the
	// creditpurchase cost-basis bucket while clearing the corresponding advance
	// receivable buckets. Otherwise we only clear receivable.
	accruedAttributable := attributed
	totalUnattributedAccrued := totalUnattributedAccruedBalance(unattributedAccrued, receivableBuckets.remainingBySpendKey)
	if accruedAttributable.GreaterThan(totalUnattributedAccrued) {
		accruedAttributable = totalUnattributedAccrued
	}

	attributions := make([]advanceAttribution, 0, len(unattributedAccrued)+len(advanceReceivables))
	if accruedAttributable.IsPositive() {
		accruedAttributions, err := allocateAccruedAttribution(currency, accruedAttributable, unattributedAccrued, receivableBuckets.remainingBySpendKey)
		if err != nil {
			return nil, err
		}

		accruedBackfillAttributions, err := allocateAccruedBackedAdvanceAttributions(accruedAttributions, unattributedAccrued, &receivableBuckets)
		if err != nil {
			return nil, err
		}

		attributions = append(attributions, accruedBackfillAttributions...)
	}

	unattributedAdvanceAmount := attributed.Sub(accruedAttributable)
	attributions = append(attributions, allocateReceivableOnlyAdvanceAttributions(unattributedAdvanceAmount, &receivableBuckets)...)

	return attributions, nil
}

// newAdvanceReceivableBuckets selects open source-less advance receivable that
// this creditpurchase is allowed to backfill. Buckets are grouped by spend charge
// for provenance matching, while the original route buckets remain ordered inside
// the group so legacy nil-spend entries cannot overwrite each other.
func newAdvanceReceivableBuckets(advanceReceivables []advanceReceivableBalance, creditFeatures []string) advanceReceivableBuckets {
	buckets := advanceReceivableBuckets{
		bySpendChargeID:     make(map[string][]advanceReceivableBalance, len(advanceReceivables)),
		remainingBySpendKey: make(map[string]alpacadecimal.Decimal, len(advanceReceivables)),
	}

	for _, advanceReceivable := range advanceReceivables {
		advanceFeatures := advanceReceivable.address.Route().Route().Features
		if !lineage.FeatureFiltersMatchAdvance(creditFeatures, advanceFeatures) {
			continue
		}

		if !advanceReceivable.amount.IsNegative() {
			continue
		}

		if _, ok := buckets.bySpendChargeID[advanceReceivable.spendChargeKey]; !ok {
			buckets.orderedSpendChargeKeys = append(buckets.orderedSpendChargeKeys, advanceReceivable.spendChargeKey)
		}

		advanceReceivable.remaining = advanceReceivable.amount.Neg()
		buckets.bySpendChargeID[advanceReceivable.spendChargeKey] = append(buckets.bySpendChargeID[advanceReceivable.spendChargeKey], advanceReceivable)
		buckets.remainingBySpendKey[advanceReceivable.spendChargeKey] = buckets.remainingBySpendKey[advanceReceivable.spendChargeKey].Add(advanceReceivable.remaining)
		buckets.total = buckets.total.Add(advanceReceivable.remaining)
	}

	slices.Sort(buckets.orderedSpendChargeKeys)

	return buckets
}

// allocateAccruedBackedAdvanceAttributions consumes receivable buckets for
// accrued value that can also be moved into the new creditpurchase cost basis.
// The accrued allocation chooses spend/tax buckets; this function maps each
// allocated spend bucket back onto the concrete receivable route buckets that
// must be cleared.
func allocateAccruedBackedAdvanceAttributions(
	accruedAllocations []currencyx.AmountAllocation[accruedBackfillBucketKey],
	unattributedAccrued []unattributedAccruedBalance,
	receivableBuckets *advanceReceivableBuckets,
) ([]advanceAttribution, error) {
	attributions := make([]advanceAttribution, 0, len(accruedAllocations))

	for _, allocation := range accruedAllocations {
		for i := range unattributedAccrued {
			if unattributedAccrued[i].key != allocation.Key {
				continue
			}

			allocated, consumed := receivableBuckets.consume(allocation.Key.spendChargeID, allocation.Amount, func(advanceReceivable advanceReceivableBalance, amount alpacadecimal.Decimal) advanceAttribution {
				return advanceAttribution{
					taxCode:         unattributedAccrued[i].taxCode,
					taxBehavior:     unattributedAccrued[i].taxBehavior,
					advanceFeatures: advanceReceivable.address.Route().Route().Features,
					spendChargeID:   advanceReceivable.spendChargeID,
					advanceAmount:   amount,
					accruedAmount:   amount,
				}
			})
			if allocation.Amount.Sub(consumed).IsPositive() {
				return nil, fmt.Errorf("advance attribution allocation %s exceeds remaining receivable for spend charge", allocation.Amount.String())
			}

			attributions = append(attributions, allocated...)
			unattributedAccrued[i].amount = unattributedAccrued[i].amount.Sub(allocation.Amount)

			break
		}
	}

	return attributions, nil
}

// allocateReceivableOnlyAdvanceAttributions consumes any remaining attributed
// amount when there is open advance receivable but no matching accrued value to
// translate. This can happen when a creditpurchase covers an advance receivable
// exposure that has already been corrected or otherwise no longer has active
// accrued balance.
func allocateReceivableOnlyAdvanceAttributions(amount alpacadecimal.Decimal, receivableBuckets *advanceReceivableBuckets) []advanceAttribution {
	if !amount.IsPositive() {
		return nil
	}

	attributions := make([]advanceAttribution, 0)
	remainingAmount := amount
	for _, spendChargeID := range receivableBuckets.orderedSpendChargeKeys {
		if !remainingAmount.IsPositive() {
			break
		}

		allocated, consumed := receivableBuckets.consume(spendChargeID, remainingAmount, func(advanceReceivable advanceReceivableBalance, amount alpacadecimal.Decimal) advanceAttribution {
			return advanceAttribution{
				advanceFeatures: advanceReceivable.address.Route().Route().Features,
				spendChargeID:   advanceReceivable.spendChargeID,
				advanceAmount:   amount,
			}
		})
		remainingAmount = remainingAmount.Sub(consumed)
		attributions = append(attributions, allocated...)
	}

	return attributions
}

// consume removes up to amount from the concrete receivable buckets for one
// spend key and builds one attribution per consumed route bucket. It is
// intentionally the only place that mutates remaining receivable state.
func (b *advanceReceivableBuckets) consume(spendChargeID string, amount alpacadecimal.Decimal, attributionFor func(advanceReceivableBalance, alpacadecimal.Decimal) advanceAttribution) ([]advanceAttribution, alpacadecimal.Decimal) {
	remainingAmount := amount
	advanceReceivables := b.bySpendChargeID[spendChargeID]
	attributions := make([]advanceAttribution, 0, len(advanceReceivables))
	consumedAmount := alpacadecimal.Zero

	for i := range advanceReceivables {
		if !remainingAmount.IsPositive() {
			break
		}

		advanceReceivable := advanceReceivables[i]
		if !advanceReceivable.remaining.IsPositive() {
			continue
		}

		advanceAmount := advanceReceivable.remaining
		if advanceAmount.GreaterThan(remainingAmount) {
			advanceAmount = remainingAmount
		}

		advanceReceivables[i].remaining = advanceReceivable.remaining.Sub(advanceAmount)
		remainingAmount = remainingAmount.Sub(advanceAmount)
		consumedAmount = consumedAmount.Add(advanceAmount)
		b.remainingBySpendKey[spendChargeID] = b.remainingBySpendKey[spendChargeID].Sub(advanceAmount)
		attributions = append(attributions, attributionFor(advanceReceivable, advanceAmount))
	}

	b.bySpendChargeID[spendChargeID] = advanceReceivables

	return attributions, consumedAmount
}

// advanceReceivableBalance is an open source-less receivable bucket that may be
// attributed to a later creditpurchase. The posting address preserves route
// dimensions, while spendChargeKey identifies which spend created the advance.
type advanceReceivableBalance struct {
	address       ledger.PostingAddress
	spendChargeID *string
	// spendChargeKey is the map key form of spendChargeID. Nil means legacy or
	// otherwise unknowable spend provenance, not a deliberate concrete charge.
	spendChargeKey string
	amount         alpacadecimal.Decimal
	remaining      alpacadecimal.Decimal
}

// advanceReceivableBalances queries balance buckets rather than sub-account
// balances because one receivable sub-account can contain multiple spend-charge
// provenance buckets. Backfill needs those buckets split so each translated
// entry preserves the spend charge that created the advance.
func (h *creditPurchaseHandler) advanceReceivableBalances(ctx context.Context, receivableAccountID models.NamespacedID, currency currencyx.Code) ([]advanceReceivableBalance, error) {
	openStatus := ledger.TransactionAuthorizationStatusOpen
	buckets, err := h.balanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: receivableAccountID.Namespace,
		Filters: ledger.Filters{
			AccountID:      &receivableAccountID.ID,
			SourceChargeID: mo.Some[*string](nil),
			SpendChargeID:  mo.None[*string](),
			Route: ledger.RouteFilter{
				Currency:                       currency,
				CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
				TransactionAuthorizationStatus: &openStatus,
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	if err != nil {
		return nil, err
	}

	out := make([]advanceReceivableBalance, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.SettledAmount.IsZero() {
			continue
		}

		spendChargeID := bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]
		out = append(out, advanceReceivableBalance{
			address:        bucket.Address,
			spendChargeID:  spendChargeID,
			spendChargeKey: lo.FromPtrOr(spendChargeID, "null"),
			amount:         bucket.SettledAmount,
		})
	}

	return out, nil
}

// unattributedAccruedBalances returns source-less, nil-cost-basis accrued value
// that can be attributed to a creditpurchase. It groups by spend charge and tax
// dimensions because cost-basis backfill must preserve both dimensions when it
// moves accrued value into the purchased source bucket.
func (h *creditPurchaseHandler) unattributedAccruedBalances(ctx context.Context, accruedAccount ledger.CustomerAccruedAccount, currency currencyx.Code) ([]unattributedAccruedBalance, error) {
	buckets, err := h.balanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: accruedAccount.ID().Namespace,
		Filters: ledger.Filters{
			AccountID:      lo.ToPtr(accruedAccount.ID().ID),
			SourceChargeID: mo.Some[*string](nil),
			Route: ledger.RouteFilter{
				Currency:  currency,
				CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	if err != nil {
		return nil, fmt.Errorf("list unattributed accrued balances: %w", err)
	}

	balancesByKey := make(map[accruedBackfillBucketKey]unattributedAccruedBalance, len(buckets))
	keys := make([]accruedBackfillBucketKey, 0, len(buckets))
	for _, bucket := range buckets {
		balance := bucket.SettledAmount
		if !balance.IsPositive() {
			continue
		}

		route := bucket.Address.Route().Route()
		spendChargeID := bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]
		key := accruedBackfillBucketKey{
			spendChargeID:   lo.FromPtrOr(spendChargeID, "null"),
			taxDimensionKey: taxDimensionRouteKey(route),
		}
		if _, ok := balancesByKey[key]; !ok {
			keys = append(keys, key)
			balancesByKey[key] = unattributedAccruedBalance{
				key:         key,
				taxCode:     route.TaxCode,
				taxBehavior: route.TaxBehavior,
			}
		}

		current := balancesByKey[key]
		current.amount = current.amount.Add(balance)
		balancesByKey[key] = current
	}

	slices.SortFunc(keys, cmpx.Compare[accruedBackfillBucketKey])

	return lo.Map(keys, func(key accruedBackfillBucketKey, _ int) unattributedAccruedBalance {
		return balancesByKey[key]
	}), nil
}

// allocateAccruedAttribution allocates a requested backfill amount across
// source-less accrued balances that have matching open advance receivable.
// This keeps the old proportional tax-bucket behavior while preserving spend
// provenance on each generated attribution leg.
func allocateAccruedAttribution(
	currency currencyx.Code,
	amount alpacadecimal.Decimal,
	unattributedAccrued []unattributedAccruedBalance,
	advanceRemainingBySpendKey map[string]alpacadecimal.Decimal,
) ([]currencyx.AmountAllocation[accruedBackfillBucketKey], error) {
	items := make([]currencyx.AmountAllocationItem[accruedBackfillBucketKey], 0, len(unattributedAccrued))
	for _, balance := range unattributedAccrued {
		remaining, ok := advanceRemainingBySpendKey[balance.key.spendChargeID]
		if !ok || !remaining.IsPositive() {
			continue
		}
		if !balance.amount.IsPositive() {
			continue
		}

		items = append(items, currencyx.AmountAllocationItem[accruedBackfillBucketKey]{
			Key:    balance.key,
			Amount: balance.amount,
		})
	}

	calculator, err := currency.Calculator()
	if err != nil {
		if err := ledger.ValidateCurrency(currency); err != nil {
			return nil, fmt.Errorf("currency: %w", err)
		}

		customCurrency, err := currencyx.NewCustomCurrency(currency, 0)
		if err != nil {
			return nil, fmt.Errorf("currency: %w", err)
		}
		calculator, err = currencyx.NewCalculator(customCurrency)
		if err != nil {
			return nil, fmt.Errorf("currency: %w", err)
		}
	}

	allocations, err := currencyx.AllocateByAmount(calculator, currencyx.AmountAllocationInput[accruedBackfillBucketKey]{
		Amount:     amount,
		Items:      items,
		CompareKey: cmpx.Compare[accruedBackfillBucketKey],
	})
	if err != nil {
		return nil, fmt.Errorf("allocate accrued attribution: %w", err)
	}

	return allocations, nil
}

// totalUnattributedAccruedBalance returns accrued capacity that has matching
// open advance receivable. This caps receivable attribution so backfill does
// not translate more accrued value than exists for eligible spend provenance.
func totalUnattributedAccruedBalance(unattributedAccrued []unattributedAccruedBalance, advanceRemainingBySpendKey map[string]alpacadecimal.Decimal) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, balance := range unattributedAccrued {
		remaining, ok := advanceRemainingBySpendKey[balance.key.spendChargeID]
		if !ok || !remaining.IsPositive() {
			continue
		}
		if balance.amount.IsPositive() {
			total = total.Add(balance.amount)
		}
	}

	return total
}

// taxDimensionRouteKey converts nullable tax route fields into comparable
// sentinel values. The sentinel is only for matching; actual posting still uses
// the route values from the hydrated balance bucket.
func taxDimensionRouteKey(route ledger.Route) taxDimensionKey {
	return taxDimensionKey{
		taxCode:     lo.FromPtrOr(route.TaxCode, "null"),
		taxBehavior: string(lo.FromPtrOr(route.TaxBehavior, "null")),
	}
}

func (k taxDimensionKey) Compare(other taxDimensionKey) int {
	if c := cmp.Compare(k.taxCode, other.taxCode); c != 0 {
		return c
	}

	return cmp.Compare(k.taxBehavior, other.taxBehavior)
}

func (k accruedBackfillBucketKey) Compare(other accruedBackfillBucketKey) int {
	if c := cmp.Compare(k.spendChargeID, other.spendChargeID); c != 0 {
		return c
	}

	return cmpx.Compare(k.taxDimensionKey, other.taxDimensionKey)
}

func (b advanceReceivableBalance) Compare(other advanceReceivableBalance) int {
	if c := cmp.Compare(b.spendChargeKey, other.spendChargeKey); c != 0 {
		return c
	}

	if c := cmpx.Compare(postingAddressRouteKeyFromAddress(b.address), postingAddressRouteKeyFromAddress(other.address)); c != 0 {
		return c
	}

	return cmp.Compare(b.address.SubAccountID(), other.address.SubAccountID())
}

// postingAddressRouteKey is the comparable subset of a posting address needed
// for deterministic helper ordering. It is not a balance key; balance matching
// is handled by the attribution keys above.
type postingAddressRouteKey struct {
	routingKey   string
	subAccountID string
}

// postingAddressRouteKeyFromAddress extracts the stable route/sub-account
// ordering fields from hydrated balance bucket addresses.
func postingAddressRouteKeyFromAddress(address ledger.PostingAddress) postingAddressRouteKey {
	return postingAddressRouteKey{
		routingKey:   address.Route().RoutingKey().Value(),
		subAccountID: address.SubAccountID(),
	}
}

func (k postingAddressRouteKey) Compare(other postingAddressRouteKey) int {
	if c := cmp.Compare(k.routingKey, other.routingKey); c != 0 {
		return c
	}

	return cmp.Compare(k.subAccountID, other.subAccountID)
}
