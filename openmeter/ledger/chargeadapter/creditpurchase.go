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

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
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
			Amount:         charge.Intent.CreditAmount,
			Currency:       charge.Intent.Currency,
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

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
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
			Amount:         charge.Intent.CreditAmount,
			Currency:       charge.Intent.Currency,
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

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)
	featureFilters := charge.Intent.FeatureFilters.Normalize()
	bookedAt := charge.Intent.ServicePeriod.To

	advanceAttributions, err := h.advanceAttributions(ctx, customerID, charge.Intent.Currency, charge.Intent.CreditAmount, featureFilters)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get advance attributions: %w", err)
	}

	advanceAttributionAmount := alpacadecimal.Zero
	for _, attribution := range advanceAttributions {
		advanceAttributionAmount = advanceAttributionAmount.Add(attribution.advanceAmount)
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceAttributionAmount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	var templates []transactions.TransactionTemplate

	for _, attribution := range advanceAttributions {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:                 bookedAt,
			Amount:             attribution.advanceAmount,
			Currency:           charge.Intent.Currency,
			CostBasis:          &costBasis,
			AdvanceFeatures:    attribution.advanceFeatures,
			AttributedFeatures: featureFilters,
			SourceChargeID:     &charge.ID,
			SpendChargeID:      attribution.spendChargeID,
		})

		if attribution.accruedAmount.IsPositive() {
			templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:             bookedAt,
				Amount:         attribution.accruedAmount,
				Currency:       charge.Intent.Currency,
				TaxCode:        attribution.taxCode,
				TaxBehavior:    attribution.taxBehavior,
				FromCostBasis:  nil,
				ToCostBasis:    &costBasis,
				SourceChargeID: &charge.ID,
				SpendChargeID:  attribution.spendChargeID,
			})
		}
	}

	if issuableAmount.IsPositive() {
		templates = append(templates, transactions.IssueCustomerReceivableTemplate{
			At:             bookedAt,
			Amount:         issuableAmount,
			Currency:       charge.Intent.Currency,
			CostBasis:      &costBasis,
			Features:       featureFilters,
			SourceChargeID: &charge.ID,
			CreditPriority: charge.Intent.Priority,
		})
	}

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		// Promotional grants settle immediately through wash so the credited FBO balance
		// does not leave an unsettled receivable behind.
		templates = append(templates,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:             bookedAt,
				Amount:         charge.Intent.CreditAmount,
				Currency:       charge.Intent.Currency,
				CostBasis:      &costBasis,
				Features:       featureFilters,
				SourceChargeID: &charge.ID,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:             bookedAt,
				Amount:         charge.Intent.CreditAmount,
				Currency:       charge.Intent.Currency,
				CostBasis:      &costBasis,
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
			Amount:            charge.Intent.CreditAmount,
			ImmediateReleases: immediateReleases,
			Currency:          charge.Intent.Currency,
			CostBasis:         &costBasis,
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

// advanceBackfillMatchKey is the projection used to pair source-less advance
// receivable with source-less accrued value. It deliberately ignores entry
// identity split fields; this operation only needs to keep spend provenance
// from drifting while it assigns the new creditpurchase source.
type advanceBackfillMatchKey struct {
	spendChargeID string
}

// accruedBackfillBucketKey adds the accrued dimensions that must remain split
// during cost-basis translation after a receivable bucket has matched by spend.
type accruedBackfillBucketKey struct {
	advanceBackfillMatchKey
	taxDimensionKey
}

// advanceAttributions determines how much of a credit purchase first covers
// existing advance receivable and accrued exposure before issuing new credit.
// It matches receivable and accrued buckets by spend charge so source attribution
// does not move value from one spending charge into another charge's provenance.
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

	calculator, err := currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	eligibleAdvanceReceivables := make([]advanceReceivableBalance, 0, len(advanceReceivables))
	advanceReceivablesByMatchKey := make(map[advanceBackfillMatchKey]advanceReceivableBalance, len(advanceReceivables))
	advanceRemainingByMatchKey := make(map[advanceBackfillMatchKey]alpacadecimal.Decimal, len(advanceReceivables))
	totalAdvanceReceivable := alpacadecimal.Zero
	for _, advanceReceivable := range advanceReceivables {
		advanceFeatures := advanceReceivable.address.Route().Route().Features
		if !lineage.FeatureFiltersMatchAdvance(creditFeatures, advanceFeatures) {
			continue
		}

		balance := advanceReceivable.amount
		if !balance.IsNegative() {
			continue
		}

		eligibleAdvanceReceivables = append(eligibleAdvanceReceivables, advanceReceivable)
		advanceReceivablesByMatchKey[advanceReceivable.matchKey] = advanceReceivable
		advanceRemainingByMatchKey[advanceReceivable.matchKey] = balance.Neg()
		totalAdvanceReceivable = totalAdvanceReceivable.Add(balance.Neg())
	}

	attributed := totalAdvanceReceivable
	if attributed.GreaterThan(amount) {
		attributed = amount
	}

	if !attributed.IsPositive() {
		return nil, nil
	}

	accruedAttributable := attributed
	totalUnattributedAccrued := totalUnattributedAccruedBalance(unattributedAccrued, advanceReceivablesByMatchKey)
	if accruedAttributable.GreaterThan(totalUnattributedAccrued) {
		accruedAttributable = totalUnattributedAccrued
	}

	attributions := make([]advanceAttribution, 0, len(unattributedAccrued)+len(advanceReceivables))
	if accruedAttributable.IsPositive() {
		accruedAttributions, err := allocateAccruedAttribution(calculator, accruedAttributable, unattributedAccrued, advanceReceivablesByMatchKey)
		if err != nil {
			return nil, err
		}

		for _, allocation := range accruedAttributions {
			for i := range unattributedAccrued {
				if unattributedAccrued[i].key != allocation.Key {
					continue
				}

				advanceReceivable := advanceReceivablesByMatchKey[allocation.Key.advanceBackfillMatchKey]
				attributions = append(attributions, advanceAttribution{
					taxCode:         unattributedAccrued[i].taxCode,
					taxBehavior:     unattributedAccrued[i].taxBehavior,
					advanceFeatures: advanceReceivable.address.Route().Route().Features,
					spendChargeID:   advanceReceivable.spendChargeID,
					advanceAmount:   allocation.Amount,
					accruedAmount:   allocation.Amount,
				})
				unattributedAccrued[i].amount = unattributedAccrued[i].amount.Sub(allocation.Amount)
				advanceRemainingByMatchKey[allocation.Key.advanceBackfillMatchKey] = advanceRemainingByMatchKey[allocation.Key.advanceBackfillMatchKey].Sub(allocation.Amount)
				break
			}
		}
	}

	unattributedAdvanceAmount := attributed.Sub(accruedAttributable)
	for _, advanceReceivable := range eligibleAdvanceReceivables {
		if !unattributedAdvanceAmount.IsPositive() {
			break
		}

		advanceAmount := advanceRemainingByMatchKey[advanceReceivable.matchKey]
		if advanceAmount.GreaterThan(unattributedAdvanceAmount) {
			advanceAmount = unattributedAdvanceAmount
		}

		if advanceAmount.IsPositive() {
			attributions = append(attributions, advanceAttribution{
				advanceFeatures: advanceReceivable.address.Route().Route().Features,
				spendChargeID:   advanceReceivable.spendChargeID,
				advanceAmount:   advanceAmount,
			})
		}

		unattributedAdvanceAmount = unattributedAdvanceAmount.Sub(advanceAmount)
	}

	return attributions, nil
}

// advanceReceivableBalance is an open source-less receivable bucket that may be
// attributed to a later creditpurchase. The posting address preserves route
// dimensions, while matchKey identifies which spend created the advance.
type advanceReceivableBalance struct {
	address       ledger.PostingAddress
	matchKey      advanceBackfillMatchKey
	spendChargeID *string
	amount        alpacadecimal.Decimal
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
		matchKey := advanceBackfillMatchKey{spendChargeID: lo.FromPtrOr(spendChargeID, "null")}
		out = append(out, advanceReceivableBalance{
			address:       bucket.Address,
			matchKey:      matchKey,
			spendChargeID: spendChargeID,
			amount:        bucket.SettledAmount,
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
		matchKey := advanceBackfillMatchKey{spendChargeID: lo.FromPtrOr(spendChargeID, "null")}
		key := accruedBackfillBucketKey{
			advanceBackfillMatchKey: matchKey,
			taxDimensionKey:         taxDimensionRouteKey(route),
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
	calculator currencyx.Calculator,
	amount alpacadecimal.Decimal,
	unattributedAccrued []unattributedAccruedBalance,
	advanceReceivablesByMatchKey map[advanceBackfillMatchKey]advanceReceivableBalance,
) ([]currencyx.AmountAllocation[accruedBackfillBucketKey], error) {
	items := make([]currencyx.AmountAllocationItem[accruedBackfillBucketKey], 0, len(unattributedAccrued))
	for _, balance := range unattributedAccrued {
		if _, ok := advanceReceivablesByMatchKey[balance.key.advanceBackfillMatchKey]; !ok {
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
func totalUnattributedAccruedBalance(unattributedAccrued []unattributedAccruedBalance, advanceReceivablesByMatchKey map[advanceBackfillMatchKey]advanceReceivableBalance) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, balance := range unattributedAccrued {
		if _, ok := advanceReceivablesByMatchKey[balance.key.advanceBackfillMatchKey]; !ok {
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

func (k advanceBackfillMatchKey) Compare(other advanceBackfillMatchKey) int {
	return cmp.Compare(k.spendChargeID, other.spendChargeID)
}

func (k accruedBackfillBucketKey) Compare(other accruedBackfillBucketKey) int {
	if c := cmpx.Compare(k.advanceBackfillMatchKey, other.advanceBackfillMatchKey); c != 0 {
		return c
	}

	return cmpx.Compare(k.taxDimensionKey, other.taxDimensionKey)
}

func (b advanceReceivableBalance) Compare(other advanceReceivableBalance) int {
	if c := cmpx.Compare(postingAddressRouteKeyFromAddress(b.address), postingAddressRouteKeyFromAddress(other.address)); c != 0 {
		return c
	}

	return cmpx.Compare(b.matchKey, other.matchKey)
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
