package credits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type BaseSuite struct {
	billingtest.BaseSuite

	Charges              charges.Service
	Ledger               ledger.Ledger
	BalanceQuerier       ledger.BalanceQuerier
	LedgerAccountService ledgeraccount.Service
	LedgerResolver       *ledgerresolvers.AccountResolver
	RevenueRecognizer    recognizer.Service
}

func (s *BaseSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())

	deps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.NoError(err)

	s.Ledger = deps.HistoricalLedger
	s.BalanceQuerier = deps.HistoricalLedger
	s.LedgerAccountService = deps.AccountService
	s.LedgerResolver = deps.ResolversService

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.NoError(err)

	revenueRecognizer, err := recognizer.NewService(recognizer.Config{
		Ledger: deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
		Lineage:            lineageService,
		TransactionManager: enttx.NewCreator(s.DBClient),
	})
	s.NoError(err)
	s.RevenueRecognizer = revenueRecognizer

	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: deps.ResolversService,
			AccountCatalog: deps.AccountService,
			BalanceQuerier: deps.HistoricalLedger,
		},
	})

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                logger,
		BillingService:        s.BillingService,
		FeatureService:        s.FeatureService,
		StreamingConnector:    s.MockStreamingConnector,
		FlatFeeHandler:        ledgerchargeadapter.NewFlatFeeHandler(deps.HistoricalLedger, transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger}, collectorService),
		CreditPurchaseHandler: ledgerchargeadapter.NewCreditPurchaseHandler(deps.HistoricalLedger, deps.HistoricalLedger, deps.ResolversService, deps.AccountService),
		UsageBasedHandler:     ledgerchargeadapter.NewUsageBasedHandler(deps.HistoricalLedger, transactions.ResolverDependencies{AccountService: deps.ResolversService, AccountCatalog: deps.AccountService, BalanceQuerier: deps.HistoricalLedger}, collectorService),
	})
	s.NoError(err)
	s.Charges = stack.ChargesService
}

func (s *BaseSuite) TearDownTest() {
	s.MockStreamingConnector.Reset()
	clock.UnFreeze()
	clock.ResetTime()
}

type CreateMockChargeIntentInput struct {
	Customer          customer.CustomerID
	Currency          currencyx.Code
	ServicePeriod     timeutil.ClosedPeriod
	Price             *productcatalog.Price
	FeatureKey        string
	Name              string
	SettlementMode    productcatalog.SettlementMode
	ManagedBy         billing.InvoiceLineManagedBy
	UniqueReferenceID string
	ProRating         productcatalog.ProRatingConfig
}

func (i *CreateMockChargeIntentInput) Validate() error {
	if i.Price == nil {
		return errors.New("price is required")
	}

	if i.Customer.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.Customer.ID == "" {
		return errors.New("customer id is required")
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	return nil
}

func (s *BaseSuite) CreateMockChargeIntent(input CreateMockChargeIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

	isFlatFee := input.Price.Type() == productcatalog.FlatPriceType
	invoiceAt := input.ServicePeriod.To

	if isFlatFee {
		price, err := input.Price.AsFlat()
		s.NoError(err)

		switch price.PaymentTerm {
		case productcatalog.InAdvancePaymentTerm:
			invoiceAt = input.ServicePeriod.From
		case productcatalog.InArrearsPaymentTerm:
			invoiceAt = input.ServicePeriod.To
		default:
			s.T().Fatalf("invalid payment term: %s", price.PaymentTerm)
		}
	}

	intentMeta := meta.Intent{
		Name:              input.Name,
		ManagedBy:         input.ManagedBy,
		ServicePeriod:     input.ServicePeriod,
		FullServicePeriod: input.ServicePeriod,
		BillingPeriod:     input.ServicePeriod,
		UniqueReferenceID: lo.EmptyableToPtr(input.UniqueReferenceID),
		CustomerID:        input.Customer.ID,
		Currency:          input.Currency,
	}

	if isFlatFee {
		price, err := input.Price.AsFlat()
		s.NoError(err)

		flatFeeIntent := flatfee.Intent{
			Intent:         intentMeta,
			PaymentTerm:    price.PaymentTerm,
			FeatureKey:     input.FeatureKey,
			InvoiceAt:      invoiceAt,
			SettlementMode: lo.CoalesceOrEmpty(input.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode),
			ProRating:      input.ProRating,

			AmountBeforeProration: price.Amount,
		}
		return charges.NewChargeIntent(flatFeeIntent)
	}

	usageBasedIntent := usagebased.Intent{
		Intent:         intentMeta,
		Price:          *input.Price,
		InvoiceAt:      invoiceAt,
		SettlementMode: lo.CoalesceOrEmpty(input.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode),
		FeatureKey:     input.FeatureKey,
	}

	return charges.NewChargeIntent(usageBasedIntent)
}

func (s *BaseSuite) CreateLedgerBackedCustomer(ns string, subjectKey string) *customer.Customer {
	s.T().Helper()

	_, err := s.LedgerResolver.EnsureBusinessAccounts(s.T().Context(), ns)
	s.NoError(err)

	cust := s.CreateTestCustomer(ns, subjectKey)
	_, err = s.LedgerResolver.CreateCustomerAccounts(s.T().Context(), cust.GetID())
	s.NoError(err)

	return cust
}

// MustCustomerFBOBalance returns customer FBO balance in a currency. Pass mo.None()
// for all cost bases, mo.Some(nil) for the explicit nil-cost-basis route, or
// mo.Some(&costBasis) for one concrete cost-basis route.
func (s *BaseSuite) MustCustomerFBOBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	return s.MustCustomerFBOBalanceWithPriority(customerID, code, costBasis, ledger.DefaultCustomerFBOPriority)
}

// MustCustomerFBOBalanceWithPriority returns customer FBO balance in a currency
// filtered by a specific credit priority. Pass mo.None() for all cost bases,
// mo.Some(nil) for the explicit nil-cost-basis route, or mo.Some(&costBasis)
// for one concrete cost-basis route.
func (s *BaseSuite) MustCustomerFBOBalanceWithPriority(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], priority int) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency:       code,
		CostBasis:      costBasis,
		CreditPriority: lo.ToPtr(priority),
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

// MustCustomerFBOBalanceForTaxCode returns customer FBO balance filtered by cost basis and tax code.
// Pass mo.None() for all tax codes, mo.Some(nil) for nil-TaxCode routes, or mo.Some(&id) for one TaxCode.
func (s *BaseSuite) MustCustomerFBOBalanceForTaxCode(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], taxCode mo.Option[*string]) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency:       code,
		CostBasis:      costBasis,
		TaxCode:        taxCode,
		CreditPriority: lo.ToPtr(ledger.DefaultCustomerFBOPriority),
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

// MustCustomerReceivableBalance returns customer receivable balance in a currency
// for one authorization state. Pass mo.None() for all cost bases, mo.Some(nil)
// for the explicit nil-cost-basis route, or mo.Some(&costBasis) for one concrete route.
func (s *BaseSuite) MustCustomerReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], status ledger.TransactionAuthorizationStatus) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: lo.ToPtr(status),
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

// MustCustomerAccruedBalance returns customer accrued balance in a currency. Pass
// mo.None() for all cost bases, mo.Some(nil) for the explicit nil-cost-basis route,
// or mo.Some(&costBasis) for one concrete cost-basis route.
func (s *BaseSuite) MustCustomerAccruedBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.AccruedAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

// MustWashBalance returns aggregate wash balance in a currency. Pass mo.None()
// for all cost bases, mo.Some(nil) for the explicit nil-cost-basis route, or
// mo.Some(&costBasis) for one concrete cost-basis route.
func (s *BaseSuite) MustWashBalance(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.WashAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *BaseSuite) MustEarningsBalance(namespace string, code currencyx.Code) alpacadecimal.Decimal {
	return s.MustEarningsBalanceForCostBasis(namespace, code, mo.None[*alpacadecimal.Decimal]())
}

// MustEarningsBalanceForCostBasis returns earnings balance in a currency. Pass
// mo.None() for all cost bases, mo.Some(nil) for the explicit nil-cost-basis route,
// or mo.Some(&costBasis) for one concrete cost-basis route.
func (s *BaseSuite) MustEarningsBalanceForCostBasis(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.EarningsAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

// MustEarningsBalanceForTaxCode returns earnings balance filtered by both cost basis and tax code.
func (s *BaseSuite) MustEarningsBalanceForTaxCode(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], taxCode mo.Option[*string]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.EarningsAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
		TaxCode:   taxCode,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

type LedgerSnapshotInput struct {
	Namespace string
	Customer  customer.CustomerID
	Currency  currencyx.Code
	CostBasis mo.Option[*alpacadecimal.Decimal]
}

type LedgerSnapshot struct {
	FBO                  alpacadecimal.Decimal
	OpenReceivable       alpacadecimal.Decimal
	AuthorizedReceivable alpacadecimal.Decimal
	Accrued              alpacadecimal.Decimal
	Wash                 alpacadecimal.Decimal
	Earnings             alpacadecimal.Decimal
}

func (s *BaseSuite) CreateLedgerSnapshot(input LedgerSnapshotInput) LedgerSnapshot {
	s.T().Helper()

	return LedgerSnapshot{
		FBO:                  s.MustCustomerFBOBalance(input.Customer, input.Currency, input.CostBasis),
		OpenReceivable:       s.MustCustomerReceivableBalance(input.Customer, input.Currency, input.CostBasis, ledger.TransactionAuthorizationStatusOpen),
		AuthorizedReceivable: s.MustCustomerReceivableBalance(input.Customer, input.Currency, input.CostBasis, ledger.TransactionAuthorizationStatusAuthorized),
		Accrued:              s.MustCustomerAccruedBalance(input.Customer, input.Currency, input.CostBasis),
		Wash:                 s.MustWashBalance(input.Namespace, input.Currency, input.CostBasis),
		Earnings:             s.MustEarningsBalance(input.Namespace, input.Currency),
	}
}

func (s *BaseSuite) AssertLedgerSnapshotUnchanged(input LedgerSnapshotInput, expected LedgerSnapshot) {
	s.T().Helper()

	s.AssertLedgerSnapshotEqual(expected, s.CreateLedgerSnapshot(input))
}

func (s *BaseSuite) AssertLedgerSnapshotEqual(expected, actual LedgerSnapshot) {
	s.T().Helper()

	s.AssertDecimalEqual(expected.FBO, actual.FBO, "FBO balance")
	s.AssertDecimalEqual(expected.OpenReceivable, actual.OpenReceivable, "open receivable balance")
	s.AssertDecimalEqual(expected.AuthorizedReceivable, actual.AuthorizedReceivable, "authorized receivable balance")
	s.AssertDecimalEqual(expected.Accrued, actual.Accrued, "accrued balance")
	s.AssertDecimalEqual(expected.Wash, actual.Wash, "wash balance")
	s.AssertDecimalEqual(expected.Earnings, actual.Earnings, "earnings balance")
}

func (s *BaseSuite) MustRecognizeRevenue(customerID customer.CustomerID, code currencyx.Code, amount alpacadecimal.Decimal) {
	s.T().Helper()

	result, err := s.RevenueRecognizer.RecognizeEarnings(s.T().Context(), recognizer.RecognizeEarningsInput{
		CustomerID: customerID,
		At:         clock.Now(),
		Currency:   code,
	})
	s.NoError(err)
	s.True(result.RecognizedAmount.Equal(amount), "recognized=%s expected=%s", result.RecognizedAmount, amount)
}

func (s *BaseSuite) MustGetChargeByID(chargeID meta.ChargeID) charges.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	s.NoError(err)

	return charge
}

type CreateCreditPurchaseIntentInput struct {
	Customer      customer.CustomerID
	Currency      currencyx.Code
	Amount        alpacadecimal.Decimal
	EffectiveAt   *time.Time
	Priority      *int
	ServicePeriod timeutil.ClosedPeriod
	Settlement    creditpurchase.Settlement
	TaxConfig     *productcatalog.TaxCodeConfig
}

func (i CreateCreditPurchaseIntentInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	if !i.Amount.IsPositive() {
		return errors.New("amount must be positive")
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if err := i.Settlement.Validate(); err != nil {
		return fmt.Errorf("settlement: %w", err)
	}

	return nil
}

func (s *BaseSuite) CreateCreditPurchaseIntent(input CreateCreditPurchaseIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

	return charges.NewChargeIntent(creditpurchase.Intent{
		Intent: meta.Intent{
			Name:              "Credit Purchase",
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        input.Customer.ID,
			Currency:          input.Currency,
			ServicePeriod:     input.ServicePeriod,
			BillingPeriod:     input.ServicePeriod,
			FullServicePeriod: input.ServicePeriod,
			TaxConfig:         input.TaxConfig,
		},
		CreditAmount: input.Amount,
		EffectiveAt:  input.EffectiveAt,
		Priority:     input.Priority,
		Settlement:   input.Settlement,
	})
}

type CreatePromotionalCreditFundingInput struct {
	Namespace string
	Customer  customer.CustomerID
	Amount    alpacadecimal.Decimal
	At        time.Time
	CostBasis alpacadecimal.Decimal
	Priority  *int
	TaxConfig *productcatalog.TaxCodeConfig
}

type CreatePromotionalCreditFundingResult struct {
	Charge         creditpurchase.Charge
	OpenReceivable alpacadecimal.Decimal
}

func (s *BaseSuite) CreatePromotionalCreditFunding(ctx context.Context, input CreatePromotionalCreditFundingInput) CreatePromotionalCreditFundingResult {
	s.T().Helper()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			s.CreateCreditPurchaseIntent(CreateCreditPurchaseIntentInput{
				Customer:      input.Customer,
				Currency:      USD,
				Amount:        input.Amount,
				Priority:      input.Priority,
				ServicePeriod: timeutil.ClosedPeriod{From: input.At, To: input.At},
				Settlement:    creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				TaxConfig:     input.TaxConfig,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	charge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)

	costBasis := mo.Some(&input.CostBasis)
	if input.Priority != nil {
		s.True(s.MustCustomerFBOBalanceWithPriority(input.Customer, USD, costBasis, *input.Priority).Equal(input.Amount))
	} else {
		s.True(s.MustCustomerFBOBalance(input.Customer, USD, costBasis).Equal(input.Amount))
	}

	return CreatePromotionalCreditFundingResult{
		Charge:         charge,
		OpenReceivable: s.MustCustomerReceivableBalance(input.Customer, USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen),
	}
}

// MustRefundCharge deletes a charge through the real refund-as-credits patch flow.
func (s *BaseSuite) MustRefundCharge(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID) {
	s.T().Helper()

	err := s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			chargeID.ID: meta.NewPatchDelete(meta.RefundAsCreditsDeletePolicy),
		},
	})
	s.NoError(err)
}
