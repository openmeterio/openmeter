package customerbalance

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ----------------------------------------------------------------------------
// Dependency interfaces
// ----------------------------------------------------------------------------

type chargesService interface {
	ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error)
}

type subAccountLister interface {
	ListSubAccounts(ctx context.Context, input ledgeraccount.ListSubAccountsInput) ([]*ledgeraccount.SubAccount, error)
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
type Service struct {
	AccountResolver   ledger.AccountResolver
	SubAccountService subAccountLister
	ChargesService    chargesService
	UsageBasedService usageBasedTotalsService

	balanceCalculator chargePendingBalanceCalculator
}

type Config struct {
	AccountResolver   ledger.AccountResolver
	SubAccountService subAccountLister
	ChargesService    chargesService
	UsageBasedService usageBasedTotalsService
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

	if c.UsageBasedService == nil {
		errs = append(errs, errors.New("usage based service is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		AccountResolver:   config.AccountResolver,
		SubAccountService: config.SubAccountService,
		ChargesService:    config.ChargesService,
		UsageBasedService: config.UsageBasedService,
		balanceCalculator: chargePendingBalanceCalculator{},
	}, nil
}

func (s *Service) GetBalance(ctx context.Context, customerID customer.CustomerID, filters ledger.RouteFilter) (ledger.Balance, error) {
	if err := s.validate(customerID, filters); err != nil {
		return nil, err
	}

	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	bookedBalance, err := customerAccounts.FBOAccount.GetBalance(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("get booked balance: %w", err)
	}

	impacts, err := s.getChargePendingBalanceImpacts(ctx, customerID, filters.Currency)
	if err != nil {
		return nil, fmt.Errorf("get charge pending balance impacts: %w", err)
	}

	return balance{
		settled: bookedBalance.Settled(),
		pending: s.balanceCalculator.CalculatePendingBalance(bookedBalance.Pending(), impacts),
	}, nil
}

func (s *Service) getFBOCurrencies(ctx context.Context, customerID customer.CustomerID) ([]currencyx.Code, error) {
	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	fboAccount, ok := customerAccounts.FBOAccount.(*ledgeraccount.CustomerFBOAccount)
	if !ok {
		return nil, fmt.Errorf("customer FBO account: unexpected type %T", customerAccounts.FBOAccount)
	}

	subAccounts, err := s.SubAccountService.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
		Namespace: fboAccount.ID().Namespace,
		AccountID: fboAccount.ID().ID,
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

func (s *Service) getChargePendingBalanceImpacts(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code) ([]Impact, error) {
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

func (s *Service) getChargePendingBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code) (*Impact, error) {
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

func (s *Service) getUsageBasedChargePendingBalanceImpact(ctx context.Context, charge charges.Charge, currency currencyx.Code) (*Impact, error) {
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

func (s *Service) validate(customerID customer.CustomerID, filters ledger.RouteFilter) error {
	var errs []error

	if s == nil {
		errs = append(errs, errors.New("service is required"))
	} else {
		if s.AccountResolver == nil {
			errs = append(errs, errors.New("account resolver is required"))
		}

		if s.SubAccountService == nil {
			errs = append(errs, errors.New("sub account service is required"))
		}

		if s.ChargesService == nil {
			errs = append(errs, errors.New("charges service is required"))
		}

		if s.UsageBasedService == nil {
			errs = append(errs, errors.New("usage based service is required"))
		}
	}

	if err := customerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if filters.Currency == "" {
		errs = append(errs, errors.New("currency filter is required"))
	}

	if _, err := filters.Normalize(); err != nil {
		errs = append(errs, fmt.Errorf("route filter: %w", err))
	}

	return errors.Join(errs...)
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
