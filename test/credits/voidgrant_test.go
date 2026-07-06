package credits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditgrant "github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestVoidGrantTestSuite(t *testing.T) {
	suite.Run(t, new(VoidGrantTestSuite))
}

type VoidGrantTestSuite struct {
	BaseSuite

	CreditVoidService  creditvoid.Service
	CreditGrantService creditgrant.Service
}

func (s *VoidGrantTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	creditVoidService, err := creditvoid.NewService(creditvoid.Config{
		Ledger: s.Ledger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: s.LedgerResolver,
			AccountCatalog: s.LedgerAccountService,
			BalanceQuerier: s.BalanceQuerier,
		},
		Breakage:           s.BreakageService,
		AccountLocker:      s.LedgerAccountService,
		TransactionManager: enttx.NewCreator(s.DBClient),
	})
	s.Require().NoError(err)
	s.CreditVoidService = creditVoidService

	creditGrantService, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: s.CreditPurchaseSvc,
		ChargesService:        s.Charges,
		BillingService:        s.BillingService,
		CustomerService:       s.CustomerService,
		CreditVoidService:     creditVoidService,
		TransactionManager:    enttx.NewCreator(s.DBClient),
	})
	s.Require().NoError(err)
	s.CreditGrantService = creditGrantService
}

func (s *VoidGrantTestSuite) TestVoidFullyUnusedGrant() {
	// given:
	// - a non-expiring promotional grant of 100 fully unused
	// when:
	// - the grant is voided
	// then:
	// - the full remaining value moves from customer FBO to breakage
	// - the grant derives as voided and the listing shows a voided transaction
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-unused")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	voidedAt := fundedAt.Add(24 * time.Hour)
	clock.FreezeTime(voidedAt)

	grant, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(grant.State.VoidedAt)
	s.Equal(voidedAt, grant.State.VoidedAt.UTC())

	s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	s.Equal(float64(100), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())

	s.requireCreditTransactionAmountsByType(cust.GetID(), nil, map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeFunded: 100,
		customerbalance.CreditTransactionTypeVoided: -100,
	})
	s.requireCreditTransactionAmountsByType(cust.GetID(), lo.ToPtr(customerbalance.CreditTransactionTypeVoided), map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeVoided: -100,
	})
}

func (s *VoidGrantTestSuite) TestVoidTwiceIsIdempotent() {
	// given:
	// - a voided grant
	// when:
	// - the void is retried
	// then:
	// - the retry succeeds with the original void time and books no extra breakage
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-twice")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	voidedAt := fundedAt.Add(time.Hour)
	clock.FreezeTime(voidedAt)

	first, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(first.State.VoidedAt)

	clock.FreezeTime(voidedAt.Add(time.Hour))

	second, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(second.State.VoidedAt)
	s.Equal(first.State.VoidedAt.UTC(), second.State.VoidedAt.UTC())

	s.Equal(float64(100), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), clock.Now()).InexactFloat64())
}

func (s *VoidGrantTestSuite) TestVoidPartiallyConsumedGrant() {
	// given:
	// - a promotional grant of 100 with 40 consumed by a credit-only usage charge
	// when:
	// - the grant is voided
	// then:
	// - only the remaining 60 moves to breakage and consumed credit stays untouched
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-partial")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	s.mustConsumeCredits(ctx, ns, cust.GetID(), 40)

	voidedAt := clock.Now().Add(time.Hour)
	clock.FreezeTime(voidedAt)

	grant, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(grant.State.VoidedAt)

	s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	s.Equal(float64(60), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())
	s.Equal(float64(40), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
}

func (s *VoidGrantTestSuite) TestVoidFullyConsumedGrantReturnsConflict() {
	// given:
	// - a promotional grant of 40 fully consumed by usage
	// when:
	// - the grant is voided
	// then:
	// - the void is rejected with a conflict because there is nothing to void
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-consumed")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(40),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	s.mustConsumeCredits(ctx, ns, cust.GetID(), 40)

	clock.FreezeTime(clock.Now().Add(time.Hour))

	_, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().Error(err)
	s.True(models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
}

func (s *VoidGrantTestSuite) TestVoidExpiringGrantReleasesFutureExpiry() {
	// given:
	// - an expiring promotional grant of 100 voided before its expiry
	// when:
	// - the original expiry time passes
	// then:
	// - expiry does not remove the same value again: breakage stays 100 and FBO 0
	// - the listing shows one voided transaction and no expired transaction
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-expiring")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	expiresAt := fundedAt.Add(30 * 24 * time.Hour)
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		ExpiresAt: &expiresAt,
		CostBasis: alpacadecimal.Zero,
	})

	voidedAt := fundedAt.Add(24 * time.Hour)
	clock.FreezeTime(voidedAt)

	grant, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(grant.State.VoidedAt)

	s.Equal(float64(100), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())

	afterExpiry := expiresAt.Add(time.Hour)
	clock.FreezeTime(afterExpiry)

	s.Equal(float64(0), s.MustCustomerFBOBalanceAsOf(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), afterExpiry).InexactFloat64())
	s.Equal(float64(100), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), afterExpiry).InexactFloat64())

	s.requireCreditTransactionAmountsByType(cust.GetID(), nil, map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeFunded: 100,
		customerbalance.CreditTransactionTypeVoided: -100,
	})
}

func (s *VoidGrantTestSuite) TestVoidAlreadyExpiredGrantReturnsConflict() {
	// given:
	// - an expiring promotional grant whose expiry already passed
	// when:
	// - the grant is voided
	// then:
	// - the void is rejected with a conflict and books nothing
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-expired")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	expiresAt := fundedAt.Add(24 * time.Hour)
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		ExpiresAt: &expiresAt,
		CostBasis: alpacadecimal.Zero,
	})

	clock.FreezeTime(expiresAt.Add(time.Hour))

	_, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().Error(err)
	s.True(models.IsGenericConflictError(err), "expected conflict error, got: %v", err)

	// Ordinary expiry still surfaces as expired, unaffected by the rejected void.
	s.requireCreditTransactionAmountsByType(cust.GetID(), nil, map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeFunded:  100,
		customerbalance.CreditTransactionTypeExpired: -100,
	})
	s.requireCreditTransactionAmountsByType(cust.GetID(), lo.ToPtr(customerbalance.CreditTransactionTypeVoided), map[customerbalance.CreditTransactionType]float64{})
}

func (s *VoidGrantTestSuite) TestVoidFeatureRestrictedGrantPreservesProvenance() {
	// given:
	// - a feature-restricted promotional grant of 100
	// when:
	// - the grant is voided
	// then:
	// - the feature-routed FBO bucket empties and breakage carries the grant's
	//   source charge provenance
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-features")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace:      ns,
		Customer:       cust.GetID(),
		Amount:         alpacadecimal.NewFromInt(100),
		At:             fundedAt,
		CostBasis:      alpacadecimal.Zero,
		FeatureFilters: creditpurchase.FeatureFilters{"api_requests_total"},
	})

	voidedAt := fundedAt.Add(time.Hour)
	clock.FreezeTime(voidedAt)

	_, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   funding.Charge.ID,
	})
	s.Require().NoError(err)

	s.Equal(float64(0), s.MustCustomerFBOBalanceForFeatures(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), mo.Some([]string{"api_requests_total"})).InexactFloat64())
	s.requireBreakageSourceBalanceBucketsAsOf(ns, ledger.RouteFilter{Currency: USD}, voidedAt, map[string]float64{
		funding.Charge.ID + "|<nil>": 100,
	})
}

func (s *VoidGrantTestSuite) TestVoidedStatusDerivationInReads() {
	// given:
	// - two grants, one of which is voided
	// then:
	// - Get returns voided_at for the voided grant only
	// - the status filter separates voided from active grants
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-status")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	voidedFunding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	keptFunding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(150),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
		Priority:  lo.ToPtr(5),
	})

	clock.FreezeTime(fundedAt.Add(time.Hour))

	_, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   voidedFunding.Charge.ID,
	})
	s.Require().NoError(err)

	voidedGrant, err := s.CreditGrantService.Get(ctx, creditgrant.GetInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   voidedFunding.Charge.ID,
	})
	s.Require().NoError(err)
	s.NotNil(voidedGrant.State.VoidedAt)

	keptGrant, err := s.CreditGrantService.Get(ctx, creditgrant.GetInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   keptFunding.Charge.ID,
	})
	s.Require().NoError(err)
	s.Nil(keptGrant.State.VoidedAt)

	s.requireGrantIDsForStatusFilter(ctx, ns, cust.ID, creditgrant.GrantStatusVoided, []string{voidedFunding.Charge.ID})

	activeIDs := s.mustListGrantIDs(ctx, ns, cust.ID, lo.ToPtr(creditgrant.GrantStatusActive))
	s.NotContains(activeIDs, voidedFunding.Charge.ID)
}

func (s *VoidGrantTestSuite) TestConcurrentVoidsDoNotDoubleBook() {
	// given:
	// - a funded grant of 100 voided concurrently from several goroutines
	// then:
	// - exactly one call books the void, the rest get a conflict, breakage totals 100
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-concurrent")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	funding := s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Amount:    alpacadecimal.NewFromInt(100),
		At:        fundedAt,
		CostBasis: alpacadecimal.Zero,
	})

	clock.FreezeTime(fundedAt.Add(time.Hour))

	errs := make([]error, 4)

	var group errgroup.Group
	for i := range errs {
		group.Go(func() error {
			_, err := s.CreditVoidService.VoidCreditPurchase(ctx, creditvoid.VoidCreditPurchaseInput{
				CustomerID: cust.GetID(),
				ChargeID:   funding.Charge.ID,
				Currency:   USD,
				Annotations: ledger.ChargeAnnotations(models.NamespacedID{
					Namespace: ns,
					ID:        funding.Charge.ID,
				}),
			})
			errs[i] = err
			return nil
		})
	}
	s.Require().NoError(group.Wait())

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		s.True(models.IsGenericConflictError(err), "losing voids must conflict, got: %v", err)
	}
	s.Equal(1, succeeded)

	s.Equal(float64(100), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), clock.Now()).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
}

func (s *VoidGrantTestSuite) setupVoidTestCustomer(ctx context.Context, ns string) *customer.Customer {
	s.T().Helper()

	s.ProvisionDefaultTaxCodes(ctx, ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	return cust
}

// mustConsumeCredits consumes credits from customer FBO by driving a
// unit-priced credit-only usage charge through its full lifecycle. It moves
// the clock past the charge's collection window.
func (s *VoidGrantTestSuite) mustConsumeCredits(ctx context.Context, ns string, customerID customer.CustomerID, units float64) {
	s.T().Helper()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	from := clock.Now()
	servicePeriod := timeutil.ClosedPeriod{
		From: from,
		To:   from.Add(7 * 24 * time.Hour),
	}

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       customerID,
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "void-test-usage",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "void-test-usage",
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.Require().NoError(err)

	s.mustAdvanceUsageBasedCharges(ctx, customerID)

	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, units, servicePeriod.From.Add(time.Hour))

	// Advance past the service period and the P2D collection window so the
	// charge finalizes and collection settles the consumed credits.
	clock.FreezeTime(servicePeriod.To.Add(12 * time.Hour))
	s.mustAdvanceUsageBasedCharges(ctx, customerID)

	clock.FreezeTime(servicePeriod.To.Add(3 * 24 * time.Hour))
	s.mustAdvanceUsageBasedCharges(ctx, customerID)
}

func (s *VoidGrantTestSuite) mustAdvanceUsageBasedCharges(ctx context.Context, customerID customer.CustomerID) {
	s.T().Helper()

	_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.Require().NoError(err)
}

func (s *VoidGrantTestSuite) requireCreditTransactionAmountsByType(customerID customer.CustomerID, txType *customerbalance.CreditTransactionType, expected map[customerbalance.CreditTransactionType]float64) {
	s.T().Helper()

	result, err := s.CustomerBalanceSvc.ListCreditTransactions(s.T().Context(), customerbalance.ListCreditTransactionsInput{
		CustomerID: customerID,
		Limit:      50,
		Type:       txType,
	})
	s.Require().NoError(err)

	actual := make(map[customerbalance.CreditTransactionType]float64, len(result.Items))
	for _, item := range result.Items {
		actual[item.Type] += item.Amount.InexactFloat64()
	}

	s.Equal(expected, actual)
}

func (s *VoidGrantTestSuite) requireGrantIDsForStatusFilter(ctx context.Context, ns, customerID string, status creditgrant.GrantStatus, expected []string) {
	s.T().Helper()

	s.Equal(expected, s.mustListGrantIDs(ctx, ns, customerID, &status))
}

func (s *VoidGrantTestSuite) mustListGrantIDs(ctx context.Context, ns, customerID string, status *creditgrant.GrantStatus) []string {
	s.T().Helper()

	result, err := s.CreditGrantService.List(ctx, creditgrant.ListInput{
		Page:       pagination.NewPage(1, 20),
		Namespace:  ns,
		CustomerID: customerID,
		Status:     status,
	})
	s.Require().NoError(err)

	return lo.Map(result.Items, func(item creditpurchase.Charge, _ int) string {
		return item.ID
	})
}
