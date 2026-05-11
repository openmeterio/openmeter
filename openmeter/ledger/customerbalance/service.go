package customerbalance

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	GetBalance(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code, query ledger.BalanceQuery) (ledger.Balance, error)
	ListCreditTransactions(ctx context.Context, input ListCreditTransactionsInput) (ListCreditTransactionsResult, error)
	GetFBOCurrencies(ctx context.Context, customerID customer.CustomerID) ([]currencyx.Code, error)
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

	balanceCalculator chargePendingBalanceCalculator
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

	return &service{
		AccountResolver:   config.AccountResolver,
		SubAccountService: config.SubAccountService,
		ChargesService:    config.ChargesService,
		CreditPurchaseSvc: config.CreditPurchaseSvc,
		UsageBasedService: config.UsageBasedService,
		Ledger:            config.Ledger,
		BalanceQuerier:    config.BalanceQuerier,
		balanceCalculator: chargePendingBalanceCalculator{},
	}, nil
}

func (s *service) GetBalance(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code, query ledger.BalanceQuery) (ledger.Balance, error) {
	query = currentBalanceQuery(query)

	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	bookedBalance, err := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency: currency,
	}, query)
	if err != nil {
		return nil, fmt.Errorf("get booked balance: %w", err)
	}

	advanceBalance, err := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.ReceivableAccount, ledger.RouteFilter{
		Currency:  currency,
		CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
	}, query)
	if err != nil {
		return nil, fmt.Errorf("get advance balance: %w", err)
	}

	// Pending balance remains a current projection from open charges.
	// Historical cursor/as-of filtering only affects the booked/settled side for now.
	impacts, err := s.getChargePendingBalanceImpacts(ctx, customerID, currency)
	if err != nil {
		return nil, fmt.Errorf("get charge pending balance impacts: %w", err)
	}

	settled := bookedBalance.Settled().Add(advanceBalance.Settled())

	return balance{
		settled: settled,
		pending: s.balanceCalculator.CalculatePendingBalance(settled, impacts),
	}, nil
}

func currentBalanceQuery(query ledger.BalanceQuery) ledger.BalanceQuery {
	if query.After != nil || query.AsOf != nil {
		return query
	}

	asOf := clock.Now()
	query.AsOf = &asOf
	return query
}

func (s *service) GetFBOCurrencies(ctx context.Context, customerID customer.CustomerID) ([]currencyx.Code, error) {
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

func (s *service) getChargePendingBalanceImpacts(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code) ([]Impact, error) {
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
		impact, err := s.getChargePendingBalanceImpact(ctx, charge, currency)
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

func (s *service) getChargePendingBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code) (*Impact, error) {
	if !chargeHasStarted(charge) {
		return nil, nil
	}

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		return getFlatFeeChargePendingBalanceImpact(charge, currency)
	case meta.ChargeTypeUsageBased:
		return s.getUsageBasedChargePendingBalanceImpact(ctx, charge, currency)
	default:
		return nil, nil
	}
}

func getFlatFeeChargePendingBalanceImpact(charge charges.Charge, currency currencyx.Code) (*Impact, error) {
	flatFeeCharge, err := charge.AsFlatFeeCharge()
	if err != nil {
		return nil, fmt.Errorf("map flat fee charge: %w", err)
	}

	if flatFeeCharge.Intent.Currency != currency {
		return nil, nil
	}

	return newImpactOrNil(charge, flatFeeCharge.State.AmountAfterProration)
}

func (s *service) getUsageBasedChargePendingBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code) (*Impact, error) {
	usageBasedCharge, err := charge.AsUsageBasedCharge()
	if err != nil {
		return nil, fmt.Errorf("map usage based charge: %w", err)
	}

	if usageBasedCharge.Intent.Currency != currency {
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

func chargeHasStarted(charge charges.Charge) bool {
	now := clock.Now()

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		if err != nil {
			return false
		}

		return !now.Before(flatFeeCharge.Intent.ServicePeriod.From)
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return false
		}

		return !now.Before(usageBasedCharge.Intent.ServicePeriod.From)
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
	pending alpacadecimal.Decimal
}

func (b balance) Settled() alpacadecimal.Decimal {
	return b.settled
}

func (b balance) Pending() alpacadecimal.Decimal {
	return b.pending
}
