package chargeadapter

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
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
		return h.issueCreditPurchaseGroup(ctx, input)
	})
}

func (h *creditPurchaseHandler) issueCreditPurchaseGroup(ctx context.Context, input chargecreditpurchase.CreditGrantInput) (chargecreditpurchase.CreditGrantResult, error) {
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
	var plan advanceBackfillPlan
	if costBasisPtr != nil {
		plan, err = h.advanceAttributions(ctx, customerID, charge.Intent.Currency, charge.Intent.CreditAmount, featureFilters, input.AdvanceLineages)
		if err != nil {
			return chargecreditpurchase.CreditGrantResult{}, fmt.Errorf("get advance attributions: %w", err)
		}
	}

	advanceAttributions := mergeAdvanceAttributions(plan.attributions)
	advanceAttributionAmount := alpacadecimal.Zero
	for _, attribution := range advanceAttributions {
		advanceAttributionAmount = advanceAttributionAmount.Add(attribution.advanceAmount)
		if attribution.collectionOriginID != nil {
			annotations[ledger.AnnotationBackfillCreditPriority] = lo.FromPtrOr(charge.Intent.Priority, ledger.DefaultCustomerFBOPriority)
		}
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceAttributionAmount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	var templates []transactions.TransactionTemplate

	for _, attribution := range advanceAttributions {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:                 advanceAttributionEffectiveAt,
			Amount:             attribution.advanceAmount,
			Currency:           charge.Intent.Currency.Reference(),
			CostBasisCurrency:  costBasisCurrency,
			CostBasis:          costBasisPtr,
			AdvanceFeatures:    attribution.advanceFeatures,
			AttributedFeatures: featureFilters,
			SourceChargeID:     &charge.ID,
			SpendChargeID:      attribution.spendChargeID,
			CollectionOriginID: attribution.collectionOriginID,
		})

		if attribution.accruedAmount.IsPositive() {
			templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:                 advanceAttributionEffectiveAt,
				Amount:             attribution.accruedAmount,
				Currency:           charge.Intent.Currency.Reference(),
				TaxCode:            attribution.taxCode,
				TaxBehavior:        attribution.taxBehavior,
				FromCostBasis:      nil,
				ToCostBasis:        costBasisPtr,
				CostBasisCurrency:  costBasisCurrency,
				SourceChargeID:     &charge.ID,
				SpendChargeID:      attribution.spendChargeID,
				CollectionOriginID: attribution.collectionOriginID,
			})
		}
	}

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
		immediateReleases := make([]breakage.PlanIssuanceImmediateRelease, 0, len(advanceAttributions))
		for _, attribution := range advanceAttributions {
			if !attribution.advanceAmount.IsPositive() {
				continue
			}

			immediateReleases = append(immediateReleases, breakage.PlanIssuanceImmediateRelease{
				Amount:             attribution.advanceAmount,
				SpendChargeID:      attribution.spendChargeID,
				CollectionOriginID: attribution.collectionOriginID,
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
		BackfillAllocations: plan.allocations,
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
	collectionOriginID *string
	taxCode            *string
	taxBehavior        *ledger.TaxBehavior
	advanceFeatures    []string
	spendChargeID      *string
	advanceAmount      alpacadecimal.Decimal
	accruedAmount      alpacadecimal.Decimal
}

// unattributedAccruedBalance is source-less accrued value available for
// creditpurchase backfill. It is keyed by the dimensions that must be preserved
// during cost-basis translation: tax treatment and spend charge provenance.
type unattributedAccruedBalance struct {
	collectionOriginID *string
	firstRecordedAt    time.Time
	key                accruedBackfillBucketKey
	taxCode            *string
	taxBehavior        *ledger.TaxBehavior
	amount             alpacadecimal.Decimal
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
	requiredFeatures []string
	bySpendChargeID  map[string][]advanceReceivableBalance
}

// advanceAttributions determines how much of a credit purchase first covers
// existing advance receivable and accrued exposure before issuing new credit.
// It matches receivable and accrued buckets by spend and collection origin so
// attribution cannot move value between independently correctable occurrences.
// Legacy rows have no spend charge; for those, route buckets still need to stay
// distinct so clearing receivable cannot accidentally net across feature routes.
func (h *creditPurchaseHandler) advanceAttributions(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencies.Currency,
	amount alpacadecimal.Decimal,
	creditFeatures []string,
	roots []legacylineage.Lineage,
) (advanceBackfillPlan, error) {
	if err := currency.Validate(); err != nil {
		return advanceBackfillPlan{}, fmt.Errorf("currency: %w", err)
	}
	currencyReference := currency.Reference()

	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return advanceBackfillPlan{}, fmt.Errorf("get customer accounts: %w", err)
	}

	advanceReceivables, err := h.advanceReceivableBalances(ctx, customerAccounts.ReceivableAccount.ID(), currencyReference)
	if err != nil {
		return advanceBackfillPlan{}, fmt.Errorf("list advance receivable balances: %w", err)
	}
	slices.SortStableFunc(advanceReceivables, cmpx.Compare[advanceReceivableBalance])

	unattributedAccrued, err := h.unattributedAccruedBalances(ctx, customerAccounts.AccruedAccount, currencyReference)
	if err != nil {
		return advanceBackfillPlan{}, err
	}

	receivableBuckets := newAdvanceReceivableBuckets(advanceReceivables, creditFeatures)

	plan := advanceBackfillPlan{}
	remaining := amount
	candidates := advanceBackfillCandidates(roots, unattributedAccrued, creditFeatures)
	for _, candidate := range candidates {
		if !remaining.IsPositive() {
			break
		}
		var spendKey string
		var accruedBuckets []unattributedAccruedBalance
		var selections []legacylineage.AdvanceBackfillAllocation
		if candidate.legacy != nil {
			root := *candidate.legacy
			receivableBuckets.requiredFeatures = root.AdvanceFeatures
			spendKey, accruedBuckets, err = h.accruedBucketsForAdvance(ctx, customerID.Namespace, root, unattributedAccrued)
			if err != nil {
				return advanceBackfillPlan{}, err
			}
			for _, segment := range root.Segments {
				selections = append(selections, legacylineage.AdvanceBackfillAllocation{SegmentID: segment.ID, Amount: segment.Amount})
			}
		} else {
			spendKey = candidate.key.spendChargeID
			for _, balance := range unattributedAccrued {
				if balance.key == candidate.key {
					accruedBuckets = append(accruedBuckets, balance)
				}
			}
			balances := receivableBuckets.bySpendChargeID[spendKey]
			if len(balances) == 0 {
				continue
			}
			receivableBuckets.requiredFeatures = balances[0].address.Route().Route().Features
			selections = []legacylineage.AdvanceBackfillAllocation{{Amount: remaining}}
		}
		for _, selection := range selections {
			if !remaining.IsPositive() {
				break
			}
			receivableCapacity := receivableBuckets.availableForSpend(spendKey)
			capacity := totalUnattributedAccruedBalance(accruedBuckets, map[string]alpacadecimal.Decimal{spendKey: receivableCapacity})
			covered := legacylineage.MinDecimal(selection.Amount, legacylineage.MinDecimal(remaining, capacity))
			if !covered.IsPositive() {
				continue
			}
			allocations, err := allocateAccruedAttribution(currency, covered, accruedBuckets, map[string]alpacadecimal.Decimal{spendKey: covered})
			if err != nil {
				return advanceBackfillPlan{}, err
			}
			attributions, err := allocateAccruedBackedAdvanceAttributions(allocations, unattributedAccrued, &receivableBuckets)
			if err != nil {
				return advanceBackfillPlan{}, err
			}
			// Keep the occurrence capacity in step with the shared balances before
			// considering another legacy segment of the same root.
			for i := range accruedBuckets {
				for _, allocation := range allocations {
					if accruedBuckets[i].key == allocation.Key {
						accruedBuckets[i].amount = accruedBuckets[i].amount.Sub(allocation.Amount)
					}
				}
			}
			plan.attributions = append(plan.attributions, attributions...)
			if candidate.legacy != nil {
				plan.allocations = append(plan.allocations, legacylineage.AdvanceBackfillAllocation{SegmentID: selection.SegmentID, Amount: covered})
			}
			remaining = remaining.Sub(covered)
		}
	}

	// Remaining receivable can be attributed even without matching accrued.
	// Only the accrued-backed amounts above become lineage backfill allocations.
	plan.attributions = append(plan.attributions, receivableBuckets.attributeRemaining(remaining)...)
	return plan, nil
}

// sortedAdvanceBackfillLineages makes FIFO independent of caller/query order.
// Collection time orders occurrences; replacement times only order segments
// within an occurrence. Copies preserve the caller's slices and omit history
// that cannot consume purchase value before any journal is loaded.
func sortedAdvanceBackfillLineages(roots []legacylineage.Lineage) []legacylineage.Lineage {
	candidates := make([]legacylineage.Lineage, 0, len(roots))
	for _, root := range roots {
		if root.OriginKind != creditrealization.LineageOriginKindAdvance {
			continue
		}
		root.Segments = lo.Filter(root.Segments, func(segment legacylineage.Segment, _ int) bool {
			return segment.State == creditrealization.LineageSegmentStateAdvanceUncovered
		})
		if len(root.Segments) == 0 {
			continue
		}
		slices.SortFunc(root.Segments, func(a, b legacylineage.Segment) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
		candidates = append(candidates, root)
	}
	slices.SortFunc(candidates, func(a, b legacylineage.Lineage) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
	})
	return candidates
}

// newAdvanceReceivableBuckets selects open source-less advance receivable that
// this creditpurchase is allowed to backfill. Buckets are grouped by spend charge
// for provenance matching, while the original route buckets remain ordered inside
// the group so legacy nil-spend entries cannot overwrite each other.
func newAdvanceReceivableBuckets(advanceReceivables []advanceReceivableBalance, creditFeatures []string) advanceReceivableBuckets {
	buckets := advanceReceivableBuckets{
		bySpendChargeID: make(map[string][]advanceReceivableBalance, len(advanceReceivables)),
	}

	for _, advanceReceivable := range advanceReceivables {
		advanceFeatures := advanceReceivable.address.Route().Route().Features
		if !legacylineage.FeatureFiltersMatchAdvance(creditFeatures, advanceFeatures) {
			continue
		}

		if !advanceReceivable.amount.IsNegative() {
			continue
		}

		advanceReceivable.remaining = advanceReceivable.amount.Neg()
		buckets.bySpendChargeID[advanceReceivable.spendChargeKey] = append(buckets.bySpendChargeID[advanceReceivable.spendChargeKey], advanceReceivable)
	}

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
					taxCode:            unattributedAccrued[i].taxCode,
					taxBehavior:        unattributedAccrued[i].taxBehavior,
					advanceFeatures:    advanceReceivable.address.Route().Route().Features,
					spendChargeID:      advanceReceivable.spendChargeID,
					collectionOriginID: advanceReceivable.collectionOriginID,
					advanceAmount:      amount,
					accruedAmount:      amount,
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

// availableForSpend applies the same feature restriction as consume, so a
// lineage occurrence cannot borrow capacity from another receivable route.
func (b *advanceReceivableBuckets) availableForSpend(spendChargeID string) alpacadecimal.Decimal {
	available := alpacadecimal.Zero
	for _, balance := range b.bySpendChargeID[spendChargeID] {
		if !slices.Equal(b.requiredFeatures, balance.address.Route().Route().Features) {
			continue
		}
		available = available.Add(balance.remaining)
	}
	return available
}

// consume removes up to amount from the concrete receivable buckets for one
// spend key and the current occurrence's feature route.
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
		if !slices.Equal(b.requiredFeatures, advanceReceivable.address.Route().Route().Features) {
			continue
		}
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
		attributions = append(attributions, attributionFor(advanceReceivable, advanceAmount))
	}

	b.bySpendChargeID[spendChargeID] = advanceReceivables

	return attributions, consumedAmount
}

// attributeRemaining uses the purchase remainder against eligible receivable
// before issuing new credit. It preserves each original spend/feature route,
// but does not imply that accrued was translated or a lineage segment funded.
func (b *advanceReceivableBuckets) attributeRemaining(amount alpacadecimal.Decimal) []advanceAttribution {
	var attributions []advanceAttribution
	for _, spendKey := range slices.Sorted(maps.Keys(b.bySpendChargeID)) {
		balances := b.bySpendChargeID[spendKey]
		for i := range balances {
			if !amount.IsPositive() {
				return attributions
			}
			balance := &balances[i]
			if !balance.remaining.IsPositive() {
				continue
			}
			attributed := legacylineage.MinDecimal(amount, balance.remaining)
			attributions = append(attributions, advanceAttribution{
				advanceFeatures:    balance.address.Route().Route().Features,
				spendChargeID:      balance.spendChargeID,
				collectionOriginID: balance.collectionOriginID,
				advanceAmount:      attributed,
			})
			balance.remaining = balance.remaining.Sub(attributed)
			amount = amount.Sub(attributed)
		}
	}
	return attributions
}

// advanceReceivableBalance is an open source-less receivable bucket that may be
// attributed to a later creditpurchase. The posting address preserves route
// dimensions, while spendChargeKey identifies which spend created the advance.
type advanceReceivableBalance struct {
	collectionOriginID *string
	address            ledger.PostingAddress
	spendChargeID      *string
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
func (h *creditPurchaseHandler) advanceReceivableBalances(ctx context.Context, receivableAccountID models.NamespacedID, currency currencies.CurrencyReference) ([]advanceReceivableBalance, error) {
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
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID, ledger.BalanceBucketGroupByCollectionOriginID},
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
			address:            bucket.Address,
			spendChargeID:      spendChargeID,
			spendChargeKey:     advanceSpendKey(spendChargeID, bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID]),
			collectionOriginID: bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID],
			amount:             bucket.SettledAmount,
		})
	}

	return out, nil
}

// unattributedAccruedBalances returns source-less, nil-cost-basis accrued value
// that can be attributed to a creditpurchase. It groups by spend charge and tax
// dimensions because cost-basis backfill must preserve both dimensions when it
// moves accrued value into the purchased source bucket.
func (h *creditPurchaseHandler) unattributedAccruedBalances(ctx context.Context, accruedAccount ledger.CustomerAccruedAccount, currency currencies.CurrencyReference) ([]unattributedAccruedBalance, error) {
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
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID, ledger.BalanceBucketGroupByCollectionOriginID},
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
			spendChargeID:   advanceSpendKey(spendChargeID, bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID]),
			taxDimensionKey: taxDimensionRouteKey(route),
		}
		if _, ok := balancesByKey[key]; !ok {
			keys = append(keys, key)
			balancesByKey[key] = unattributedAccruedBalance{
				key:                key,
				collectionOriginID: bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID],
				firstRecordedAt:    bucket.FirstRecordedAt,
				taxCode:            route.TaxCode,
				taxBehavior:        route.TaxBehavior,
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
	calculator currencyx.Currency,
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
	bySpend := make(map[string]alpacadecimal.Decimal)
	for _, balance := range unattributedAccrued {
		if balance.amount.IsPositive() {
			bySpend[balance.key.spendChargeID] = bySpend[balance.key.spendChargeID].Add(balance.amount)
		}
	}
	total := alpacadecimal.Zero
	for spend, accrued := range bySpend {
		total = total.Add(legacylineage.MinDecimal(accrued, advanceRemainingBySpendKey[spend]))
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

type advanceBackfillPlan struct {
	attributions []advanceAttribution
	allocations  []legacylineage.AdvanceBackfillAllocation
}

// Multiple collection occurrences can share posting routes. Coalesce their
// legs after FIFO selection, keeping lineage allocations occurrence-specific.
func mergeAdvanceAttributions(attributions []advanceAttribution) []advanceAttribution {
	out := make([]advanceAttribution, 0, len(attributions))
	for _, attribution := range attributions {
		idx := slices.IndexFunc(out, attribution.canMergeInto)
		if idx == -1 {
			out = append(out, attribution)
			continue
		}
		out[idx].advanceAmount = out[idx].advanceAmount.Add(attribution.advanceAmount)
		out[idx].accruedAmount = out[idx].accruedAmount.Add(attribution.accruedAmount)
	}
	return out
}

// canMergeInto preserves the tax route of accrued. A receivable-only remainder
// can join an existing leg regardless of tax, since it adds no accrued value.
// The remainder must not add a duplicate receivable leg that makes corrections ambiguous.
func (a advanceAttribution) canMergeInto(other advanceAttribution) bool {
	if lo.FromPtr(a.collectionOriginID) != lo.FromPtr(other.collectionOriginID) {
		return false
	}
	if lo.FromPtr(a.spendChargeID) != lo.FromPtr(other.spendChargeID) || !slices.Equal(a.advanceFeatures, other.advanceFeatures) {
		return false
	}
	return !a.accruedAmount.IsPositive() ||
		(lo.FromPtr(a.taxCode) == lo.FromPtr(other.taxCode) && lo.FromPtr(a.taxBehavior) == lo.FromPtr(other.taxBehavior))
}

func (h *creditPurchaseHandler) accruedBucketsForAdvance(ctx context.Context, namespace string, root legacylineage.Lineage, balances []unattributedAccruedBalance) (string, []unattributedAccruedBalance, error) {
	if root.OriginalTransactionGroupID == "" {
		return "", nil, fmt.Errorf("advance lineage %s is missing its original transaction group", root.ID)
	}
	group, err := h.ledger.GetTransactionGroup(ctx, models.NamespacedID{Namespace: namespace, ID: root.OriginalTransactionGroupID})
	if err != nil {
		return "", nil, err
	}
	key, err := originalAdvanceAccruedBucket(group)
	if err != nil {
		return "", nil, err
	}
	var matched []unattributedAccruedBalance
	for _, balance := range balances {
		if balance.key == key {
			matched = append(matched, balance)
		}
	}
	return key.spendChargeID, matched, nil
}

// originalAdvanceAccruedBucket reads the original tax route and spend provenance,
// including nil-spend legacy collections, rather than inferring them from charge metadata.
func originalAdvanceAccruedBucket(group ledger.TransactionGroup) (accruedBackfillBucketKey, error) {
	for _, tx := range group.Transactions() {
		code, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		if err != nil {
			return accruedBackfillBucketKey{}, err
		}
		if code != transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			continue
		}
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsPositive() {
				continue
			}
			return accruedBackfillBucketKey{
				spendChargeID:   lo.FromPtrOr(entry.SpendChargeID(), "null"),
				taxDimensionKey: taxDimensionRouteKey(entry.PostingAddress().Route().Route()),
			}, nil
		}
	}
	return accruedBackfillBucketKey{}, fmt.Errorf("original advance collection missing from group %s", group.ID().ID)
}

// advanceSpendKey prevents two runs of the same charge from sharing attribution.
func advanceSpendKey(spend, origin *string) string {
	if origin == nil {
		return lo.FromPtrOr(spend, "null")
	}
	return lo.FromPtrOr(spend, "null") + ":" + *origin
}

// advanceBackfillCandidate is transient selection data. Origin capacity comes
// from journal sums; only legacy candidates carry persisted segment state.
type advanceBackfillCandidate struct {
	recordedAt time.Time
	id         string
	key        accruedBackfillBucketKey
	legacy     *legacylineage.Lineage
}

func advanceBackfillCandidates(roots []legacylineage.Lineage, balances []unattributedAccruedBalance, features []string) []advanceBackfillCandidate {
	var candidates []advanceBackfillCandidate
	for _, root := range sortedAdvanceBackfillLineages(legacylineage.FilterAdvanceLineagesForBackfill(roots, features)) {
		candidates = append(candidates, advanceBackfillCandidate{recordedAt: root.CreatedAt, id: root.ID, legacy: &root})
	}
	for _, balance := range balances {
		if balance.collectionOriginID == nil || !balance.amount.IsPositive() {
			continue
		}
		candidates = append(candidates, advanceBackfillCandidate{recordedAt: balance.firstRecordedAt, id: *balance.collectionOriginID, key: balance.key})
	}
	slices.SortFunc(candidates, func(a, b advanceBackfillCandidate) int {
		return cmp.Or(a.recordedAt.Compare(b.recordedAt), cmp.Compare(a.id, b.id), cmpx.Compare(a.key, b.key))
	})
	return candidates
}
