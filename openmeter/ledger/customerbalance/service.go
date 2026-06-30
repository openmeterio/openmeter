package customerbalance

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	GetBalance(ctx context.Context, input GetBalanceServiceInput) (Balance, error)
	GetSettledBalance(ctx context.Context, input GetBalanceServiceInput) (alpacadecimal.Decimal, error)
	ListCreditTransactions(ctx context.Context, input ListCreditTransactionsInput) (ListCreditTransactionsResult, error)
	GetBalanceCurrencies(ctx context.Context, input GetBalanceCurrenciesInput) ([]currencyx.Code, error)
}

type Balance interface {
	Settled() alpacadecimal.Decimal
	Live() alpacadecimal.Decimal
	Pending() alpacadecimal.Decimal
}

// ----------------------------------------------------------------------------
// Dependency interfaces
// ----------------------------------------------------------------------------

type chargesService interface {
	GetByIDs(ctx context.Context, input charges.GetByIDsInput) (charges.Charges, error)
	ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error)
}

type creditPurchaseActivityService interface {
	ListFundedCreditActivities(ctx context.Context, input creditpurchase.ListFundedCreditActivitiesInput) (creditpurchase.ListFundedCreditActivitiesResult, error)
}

type subAccountLister interface {
	ListSubAccounts(ctx context.Context, input ledger.ListSubAccountsInput) ([]ledger.SubAccount, error)
}

type usageBasedTotalsService interface {
	GetCurrentTotals(ctx context.Context, input usagebased.GetCurrentTotalsInput) (usagebased.GetCurrentTotalsResult, error)
}

const chargeListPageSize = 100

// ----------------------------------------------------------------------------
// Service
// ----------------------------------------------------------------------------

// service is NOT the RTE (Real Time Engine)
// - it is a simple service to bridge the gap until we get to implementing the RTE
// - this should be used for balance queries until the RTE is implemented
type service struct {
	AccountResolver   ledger.AccountResolver
	SubAccountService subAccountLister
	ChargesService    chargesService
	CreditPurchaseSvc creditPurchaseActivityService
	UsageBasedService usageBasedTotalsService
	Ledger            ledger.Ledger
	BalanceQuerier    ledger.BalanceQuerier
	Breakage          ledgerbreakage.Service

	balanceCalculator chargeLiveBalanceCalculator
}

var _ Service = (*service)(nil)

type Config struct {
	AccountResolver   ledger.AccountResolver
	SubAccountService subAccountLister
	ChargesService    chargesService
	CreditPurchaseSvc creditPurchaseActivityService
	UsageBasedService usageBasedTotalsService
	Ledger            ledger.Ledger
	BalanceQuerier    ledger.BalanceQuerier
	Breakage          ledgerbreakage.Service
}

type GetBalanceServiceInput struct {
	CustomerID    customer.CustomerID
	Currency      currencyx.Code
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
	BalanceQuery  ledger.BalanceQuery
}

type GetBalanceCurrenciesInput struct {
	CustomerID    customer.CustomerID
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
	AsOf          *time.Time
}

func (i GetBalanceServiceInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := ValidateFeatureFilter(i.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	if i.BalanceQuery.After != nil {
		if err := i.BalanceQuery.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("balance query after: %w", err))
		}
	}

	if i.BalanceQuery.AsOf != nil && i.BalanceQuery.AsOf.IsZero() {
		errs = append(errs, errors.New("balance query asOf must not be zero"))
	}

	if i.BalanceQuery.After != nil && i.BalanceQuery.AsOf != nil {
		errs = append(errs, errors.New("balance query after and asOf cannot both be set"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i GetBalanceServiceInput) balanceQuery() ledger.BalanceQuery {
	query := i.BalanceQuery
	if query.After != nil || query.AsOf != nil {
		return query
	}

	asOf := clock.Now()
	query.AsOf = &asOf
	return query
}

func (i GetBalanceServiceInput) pendingGrantAsOf() time.Time {
	if i.BalanceQuery.AsOf != nil {
		return *i.BalanceQuery.AsOf
	}

	return clock.Now()
}

func (i GetBalanceCurrenciesInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := ValidateFeatureFilter(i.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, errors.New("asOf must not be zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i GetBalanceCurrenciesInput) pendingGrantAsOf() time.Time {
	if i.AsOf != nil {
		return *i.AsOf
	}

	return clock.Now()
}

func (i GetBalanceServiceInput) bookedRoute() ledger.RouteFilter {
	route := i.featureRoute()
	route.Currency = i.Currency

	return route
}

func (i GetBalanceServiceInput) advanceRoute() ledger.RouteFilter {
	route := i.featureRoute()
	route.Currency = i.Currency
	route.CostBasis = mo.Some[*alpacadecimal.Decimal](nil)

	return route
}

func (i GetBalanceServiceInput) featureRoute() ledger.RouteFilter {
	return featureFilterRoute(normalizeFeatureFilter(i.FeatureFilter))
}

func (c Config) Validate() error {
	var errs []error

	if c.AccountResolver == nil {
		errs = append(errs, errors.New("account resolver is required"))
	}

	if c.SubAccountService == nil {
		errs = append(errs, errors.New("sub account service is required"))
	}

	if c.ChargesService == nil {
		errs = append(errs, errors.New("charges service is required"))
	}

	if c.CreditPurchaseSvc == nil {
		errs = append(errs, errors.New("credit purchase service is required"))
	}

	if c.UsageBasedService == nil {
		errs = append(errs, errors.New("usage based service is required"))
	}

	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}

	if c.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	breakageService := config.Breakage
	if breakageService == nil {
		breakageService = ledgerbreakage.NewNoopService()
	}

	return &service{
		AccountResolver:   config.AccountResolver,
		SubAccountService: config.SubAccountService,
		ChargesService:    config.ChargesService,
		CreditPurchaseSvc: config.CreditPurchaseSvc,
		UsageBasedService: config.UsageBasedService,
		Ledger:            config.Ledger,
		BalanceQuerier:    config.BalanceQuerier,
		Breakage:          breakageService,
		balanceCalculator: chargeLiveBalanceCalculator{},
	}, nil
}

func (s *service) GetBalance(ctx context.Context, input GetBalanceServiceInput) (Balance, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	settled, err := s.getSettledBalance(ctx, input)
	if err != nil {
		return nil, err
	}

	// Live balance remains a current projection from open charges.
	// Historical cursor/as-of filtering only affects the booked/available side for now.
	impacts, err := s.getChargeLiveBalanceImpacts(ctx, input.CustomerID, input.Currency, normalizeFeatureFilter(input.FeatureFilter))
	if err != nil {
		return nil, fmt.Errorf("get charge live balance impacts: %w", err)
	}

	live, err := s.calculateLiveBalance(ctx, input, settled, impacts)
	if err != nil {
		return nil, fmt.Errorf("calculate live balance: %w", err)
	}

	pending, err := s.getPendingGrantAmount(ctx, input.CustomerID, input.Currency, normalizeFeatureFilter(input.FeatureFilter), input.pendingGrantAsOf())
	if err != nil {
		return nil, fmt.Errorf("get pending grant amount: %w", err)
	}

	return balance{
		settled: settled,
		live:    live,
		pending: pending,
	}, nil
}

type liveBalanceSource struct {
	route  ledger.Route
	amount alpacadecimal.Decimal
	cursor string
}

var _ cmpx.Comparable[liveBalanceSource] = liveBalanceSource{}

func (s *service) calculateLiveBalance(ctx context.Context, input GetBalanceServiceInput, settled alpacadecimal.Decimal, impacts []Impact) (alpacadecimal.Decimal, error) {
	// Live charge impacts must be applied against the same credit sources the
	// collector could actually consume. An aggregate settled balance would let a
	// charge for one feature reduce credit restricted to another feature, even
	// though the eventual ledger collection would leave that credit untouched.
	sources, err := s.getLiveBalanceSources(ctx, input)
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return s.balanceCalculator.CalculateLiveBalanceFromSources(settled, sources, impacts), nil
}

func (s *service) getLiveBalanceSources(ctx context.Context, input GetBalanceServiceInput) ([]liveBalanceSource, error) {
	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	subAccounts, err := s.SubAccountService.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: customerAccounts.FBOAccount.ID().Namespace,
		AccountID: customerAccounts.FBOAccount.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub accounts: %w", err)
	}

	routeFilter := input.bookedRoute()
	query := input.balanceQuery()
	sources := make([]liveBalanceSource, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if !route.Matches(routeFilter) {
			continue
		}

		sourceBalance, err := s.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, query)
		if err != nil {
			return nil, fmt.Errorf("get sub account balance: %w", err)
		}

		if !sourceBalance.IsPositive() {
			continue
		}

		sources = append(sources, liveBalanceSource{
			route:  route,
			amount: sourceBalance,
			cursor: subAccount.Address().SubAccountID(),
		})
	}

	slices.SortStableFunc(sources, cmpx.Compare[liveBalanceSource])

	return sources, nil
}

// Compare keeps the source walk aligned with collection order. This matters for
// live balance because source amounts are consumed in-memory as impacts are
// applied, so a shared unrestricted source exhausted by one impact must not be
// counted again for a later impact.
func (s liveBalanceSource) Compare(other liveBalanceSource) int {
	if c := cmp.Compare(lo.FromPtrOr(s.route.CreditPriority, ledger.DefaultCustomerFBOPriority), lo.FromPtrOr(other.route.CreditPriority, ledger.DefaultCustomerFBOPriority)); c != 0 {
		return c
	}

	leftRestricted := len(s.route.Features) > 0
	rightRestricted := len(other.route.Features) > 0
	if leftRestricted != rightRestricted {
		if leftRestricted {
			return -1
		}

		return 1
	}

	return cmp.Compare(s.cursor, other.cursor)
}

func (s *service) GetSettledBalance(ctx context.Context, input GetBalanceServiceInput) (alpacadecimal.Decimal, error) {
	if err := input.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	return s.getSettledBalance(ctx, input)
}

func (s *service) getSettledBalance(ctx context.Context, input GetBalanceServiceInput) (alpacadecimal.Decimal, error) {
	query := input.balanceQuery()

	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("get customer accounts: %w", err)
	}

	bookedBalance, err := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.FBOAccount, input.bookedRoute(), query)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("get booked balance: %w", err)
	}

	advanceBalance, err := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.ReceivableAccount, input.advanceRoute(), query)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("get advance balance: %w", err)
	}

	return bookedBalance.Add(advanceBalance), nil
}

func (s *service) GetBalanceCurrencies(ctx context.Context, input GetBalanceCurrenciesInput) ([]currencyx.Code, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// FIXME[RTE]: when GetBalances discovers currencies, pending grants are
	// scanned here and then scanned again once per currency in GetBalance. This
	// is accepted as temporary bridge behavior until scheduled grants have an
	// RTE-owned fact/index or a shared candidate cache in this service.
	fboCurrencies, err := s.getFBOCurrencies(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	pendingCurrencies, err := s.getPendingGrantCurrencies(ctx, input.CustomerID, normalizeFeatureFilter(input.FeatureFilter), input.pendingGrantAsOf())
	if err != nil {
		return nil, err
	}

	return dedupeCurrencies(append(fboCurrencies, pendingCurrencies...)), nil
}

func (s *service) getFBOCurrencies(ctx context.Context, customerID customer.CustomerID) ([]currencyx.Code, error) {
	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	subAccounts, err := s.SubAccountService.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: customerAccounts.FBOAccount.ID().Namespace,
		AccountID: customerAccounts.FBOAccount.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub accounts: %w", err)
	}

	seen := make(map[currencyx.Code]struct{}, len(subAccounts))
	codes := make([]currencyx.Code, 0, len(subAccounts))

	for _, sa := range subAccounts {
		c := sa.Route().Currency
		if _, ok := seen[c]; ok {
			continue
		}

		seen[c] = struct{}{}
		codes = append(codes, c)
	}

	return codes, nil
}

func (s *service) getPendingGrantCurrencies(
	ctx context.Context,
	customerID customer.CustomerID,
	featureFilter mo.Option[creditpurchase.FeatureFilters],
	asOf time.Time,
) ([]currencyx.Code, error) {
	charges, err := s.listPendingGrantCandidateCharges(ctx, customerID)
	if err != nil {
		return nil, err
	}

	codes := make([]currencyx.Code, 0, len(charges))
	for _, charge := range charges {
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		if err != nil {
			return nil, fmt.Errorf("map credit purchase charge: %w", err)
		}

		if !isPendingCreditGrantAt(creditPurchaseCharge, asOf) {
			continue
		}

		if !featureFilterMatchesCreditPurchase(featureFilter, creditPurchaseCharge.Intent.FeatureFilters) {
			continue
		}

		codes = append(codes, creditPurchaseCharge.Intent.Currency)
	}

	return dedupeCurrencies(codes), nil
}

func (s *service) getPendingGrantAmount(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	featureFilter mo.Option[creditpurchase.FeatureFilters],
	asOf time.Time,
) (alpacadecimal.Decimal, error) {
	charges, err := s.listPendingGrantCandidateCharges(ctx, customerID)
	if err != nil {
		return alpacadecimal.Zero, err
	}

	total := alpacadecimal.Zero
	for _, charge := range charges {
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		if err != nil {
			return alpacadecimal.Zero, fmt.Errorf("map credit purchase charge: %w", err)
		}

		if creditPurchaseCharge.Intent.Currency != currency {
			continue
		}

		if !isPendingCreditGrantAt(creditPurchaseCharge, asOf) {
			continue
		}

		if !featureFilterMatchesCreditPurchase(featureFilter, creditPurchaseCharge.Intent.FeatureFilters) {
			continue
		}

		total = total.Add(creditPurchaseCharge.Intent.CreditAmount)
	}

	return total, nil
}

func (s *service) listPendingGrantCandidateCharges(ctx context.Context, customerID customer.CustomerID) ([]charges.Charge, error) {
	// FIXME[RTE]: this is terrible and too slow. It expands and scans
	// credit-purchase charges on every balance read until pending scheduled
	// grants have a durable query shape. Keep query-side heuristics conservative:
	// only exclude charges that definitely cannot become ledger credit, and let
	// isPendingCreditGrantAt handle lifecycle edge cases like final realized
	// grants booked in the future.
	items, err := pagination.CollectAll(
		ctx,
		pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[charges.Charge], error) {
			return s.ChargesService.ListCharges(ctx, charges.ListChargesInput{
				Page:        page,
				Namespace:   customerID.Namespace,
				CustomerIDs: []string{customerID.ID},
				ChargeTypes: []meta.ChargeType{
					meta.ChargeTypeCreditPurchase,
				},
				StatusNotIn: []meta.ChargeStatus{
					meta.ChargeStatusDeleted,
				},
				Expands: meta.Expands{meta.ExpandRealizations},
			})
		}),
		chargeListPageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("list credit purchase charges: %w", err)
	}

	return items, nil
}

// isPendingCreditGrantAt reports whether a credit purchase should contribute to
// the scheduled-grant pending amount at asOf.
//
// A charge is pending only while it can still legitimately become effective
// ledger credit and either has no credit grant realization yet, or has already
// been realized with a future ledger booking time.
func isPendingCreditGrantAt(charge creditpurchase.Charge, asOf time.Time) bool {
	if !canBecomeEffectiveLedgerCreditAt(charge, asOf) {
		return false
	}

	if charge.Realizations.CreditGrantRealization == nil {
		return true
	}

	return charge.Intent.ServicePeriod.To.After(asOf)
}

func canBecomeEffectiveLedgerCreditAt(charge creditpurchase.Charge, asOf time.Time) bool {
	if charge.CreatedAt.After(asOf) {
		return false
	}

	if charge.Status == creditpurchase.StatusDeleted || charge.IsDeletedAt(asOf) {
		return false
	}

	if charge.Realizations.InvoiceSettlement != nil && charge.Realizations.InvoiceSettlement.IsDeletedAt(asOf) {
		return false
	}

	if charge.Realizations.ExternalPaymentSettlement != nil && charge.Realizations.ExternalPaymentSettlement.IsDeletedAt(asOf) {
		return false
	}

	// CreditGrantRealization is a successful ledger transaction reference, not a
	// realization state machine. Failed grant writes leave it unset; voided
	// settlement paths are represented by deleted charge/payment realizations
	// above.
	if charge.Status == creditpurchase.StatusFinal && charge.Realizations.CreditGrantRealization == nil {
		return false
	}

	return true
}

func featureFilterMatchesCreditPurchase(featureFilter mo.Option[creditpurchase.FeatureFilters], grantFeatures creditpurchase.FeatureFilters) bool {
	if featureFilter.IsAbsent() {
		return true
	}

	grantFeatures = grantFeatures.Normalize()
	filterFeatures := featureFilter.OrEmpty()
	if filterFeatures == nil {
		return len(grantFeatures) == 0
	}

	if len(grantFeatures) == 0 {
		return true
	}

	return len(filterFeatures) == 1 && slices.Contains(grantFeatures, filterFeatures[0])
}

func (s *service) getChargeLiveBalanceImpacts(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code, featureFilter mo.Option[creditpurchase.FeatureFilters]) ([]Impact, error) {
	items, err := pagination.CollectAll(
		ctx,
		pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[charges.Charge], error) {
			return s.ChargesService.ListCharges(ctx, charges.ListChargesInput{
				Page:        page,
				Namespace:   customerID.Namespace,
				CustomerIDs: []string{customerID.ID},
				ChargeTypes: []meta.ChargeType{
					meta.ChargeTypeFlatFee,
					meta.ChargeTypeUsageBased,
				},
				StatusNotIn: []meta.ChargeStatus{meta.ChargeStatusFinal},
				Expands:     meta.Expands{meta.ExpandRealizations},
			})
		}),
		chargeListPageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("list charges: %w", err)
	}

	impacts := make([]Impact, 0, len(items))
	for _, charge := range items {
		impact, err := s.getChargeLiveBalanceImpact(ctx, charge, currency, featureFilter)
		if err != nil {
			return nil, err
		}

		if impact == nil {
			continue
		}

		impacts = append(impacts, *impact)
	}

	return impacts, nil
}

func (s *service) getChargeLiveBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code, featureFilter mo.Option[creditpurchase.FeatureFilters]) (*Impact, error) {
	if !chargeHasStarted(charge) {
		return nil, nil
	}

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		return getFlatFeeChargePendingBalanceImpact(charge, currency, featureFilter)
	case meta.ChargeTypeUsageBased:
		return s.getUsageBasedChargePendingBalanceImpact(ctx, charge, currency, featureFilter)
	default:
		return nil, nil
	}
}

func getFlatFeeChargePendingBalanceImpact(charge charges.Charge, currency currencyx.Code, featureFilter mo.Option[creditpurchase.FeatureFilters]) (*Impact, error) {
	flatFeeCharge, err := charge.AsFlatFeeCharge()
	if err != nil {
		return nil, fmt.Errorf("map flat fee charge: %w", err)
	}

	if flatFeeCharge.Intent.GetCurrency() != currency {
		return nil, nil
	}

	if !featureFilterMatchesChargeFeatureKey(featureFilter, flatFeeCharge.Intent.GetEffectiveFeatureKey()) {
		return nil, nil
	}

	return newImpactOrNil(charge, flatFeeCharge.State.AmountAfterProration)
}

func (s *service) getUsageBasedChargePendingBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code, featureFilter mo.Option[creditpurchase.FeatureFilters]) (*Impact, error) {
	usageBasedCharge, err := charge.AsUsageBasedCharge()
	if err != nil {
		return nil, fmt.Errorf("map usage based charge: %w", err)
	}

	if usageBasedCharge.Intent.GetCurrency() != currency {
		return nil, nil
	}

	if !featureFilterMatchesChargeFeatureKey(featureFilter, usageBasedCharge.Intent.GetEffectiveFeatureKey()) {
		return nil, nil
	}

	currentTotals, err := s.UsageBasedService.GetCurrentTotals(ctx, usagebased.GetCurrentTotalsInput{
		ChargeID: usageBasedCharge.GetChargeID(),
	})
	if err != nil {
		return nil, fmt.Errorf("get current totals for charge %s: %w", usageBasedCharge.ID, err)
	}

	return newImpactOrNil(charges.NewCharge(currentTotals.Charge), currentTotals.DueTotals.Total)
}

// featureFilterMatchesChargeFeatureKey is query-scope matching: a feature
// balance view includes unrestricted charge impacts so it can show the customer's
// shared-credit exposure for that feature. Actual credit allocability is checked
// separately when live impacts are applied to concrete credit sources.
func featureFilterMatchesChargeFeatureKey(featureFilter mo.Option[creditpurchase.FeatureFilters], featureKey string) bool {
	if featureFilter.IsAbsent() {
		return true
	}

	features := featureFilter.OrEmpty()
	if features == nil {
		return featureKey == ""
	}

	if featureKey == "" {
		return true
	}

	return len(features) == 1 && features[0] == featureKey
}

func chargeHasStarted(charge charges.Charge) bool {
	now := clock.Now()

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		if err != nil {
			return false
		}

		return !now.Before(flatFeeCharge.Intent.GetEffectiveServicePeriod().From)
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return false
		}

		return !now.Before(usageBasedCharge.Intent.GetEffectiveServicePeriod().From)
	default:
		return false
	}
}

func newImpactOrNil(charge charges.Charge, amount alpacadecimal.Decimal) (*Impact, error) {
	impact, err := NewImpact(charge, amount)
	if err != nil {
		return nil, err
	}

	if impact.OutstandingAmount().IsZero() {
		return nil, nil
	}

	return &impact, nil
}

type balance struct {
	settled alpacadecimal.Decimal
	live    alpacadecimal.Decimal
	pending alpacadecimal.Decimal
}

func (b balance) Settled() alpacadecimal.Decimal {
	return b.settled
}

func (b balance) Live() alpacadecimal.Decimal {
	return b.live
}

func (b balance) Pending() alpacadecimal.Decimal {
	return b.pending
}
