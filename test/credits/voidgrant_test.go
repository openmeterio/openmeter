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

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
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

	CreditGrantService creditgrant.Service
}

func (s *VoidGrantTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	creditGrantService, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: s.CreditPurchaseSvc,
		ChargesService:        s.Charges,
		BillingService:        s.BillingService,
		CustomerService:       s.CustomerService,
		CreditVoidService:     s.CreditVoidService,
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
	// - the full remaining value moves from customer FBO back to open receivable
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
	s.Equal(float64(100), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())

	s.requireCreditTransactionAmountsByType(cust.GetID(), nil, map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeFunded: 100,
		customerbalance.CreditTransactionTypeVoided: -100,
	})
	s.requireCreditTransactionAmountsByType(cust.GetID(), lo.ToPtr(customerbalance.CreditTransactionTypeVoided), map[customerbalance.CreditTransactionType]float64{
		customerbalance.CreditTransactionTypeVoided: -100,
	})

	voidedImpacts, err := s.CreditVoidService.ListVoidedCreditImpacts(ctx, creditvoid.ListVoidedCreditImpactsInput{
		CustomerID: cust.GetID(),
		Currency:   lo.ToPtr(USD),
		AsOf:       voidedAt.Add(time.Second),
		Limit:      10,
	})
	s.Require().NoError(err)
	s.Require().Len(voidedImpacts.Items, 1)
	s.Equal(string(transactions.TemplateCodeIssueCustomerReceivable), voidedImpacts.Items[0].Annotations[ledger.AnnotationTransactionTemplateCode])
	s.Equal(string(ledger.TransactionDirectionCorrection), voidedImpacts.Items[0].Annotations[ledger.AnnotationTransactionDirection])
}

func (s *VoidGrantTestSuite) TestVoidTwiceIsIdempotent() {
	// given:
	// - a voided grant
	// when:
	// - the void is retried
	// then:
	// - the retry succeeds with the original void time and books no extra ledger movement
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

	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), clock.Now()).InexactFloat64())
	s.Equal(float64(100), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
}

func (s *VoidGrantTestSuite) TestVoidPartiallyConsumedGrant() {
	// given:
	// - a promotional grant of 100 with 40 consumed by a credit-only usage charge
	// when:
	// - the grant is voided
	// then:
	// - only the remaining 60 moves back to receivable and consumed credit stays untouched
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
	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())
	s.Equal(float64(40), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	s.Equal(float64(60), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
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
	// - expiry does not remove the same value again: breakage stays 0 and FBO 0
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

	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())
	s.Equal(float64(100), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())

	afterExpiry := expiresAt.Add(time.Hour)
	clock.FreezeTime(afterExpiry)

	s.Equal(float64(0), s.MustCustomerFBOBalanceAsOf(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), afterExpiry).InexactFloat64())
	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), afterExpiry).InexactFloat64())

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
	// - the feature-routed FBO bucket empties and the void transaction carries
	//   the grant's source charge provenance
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
	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), voidedAt).InexactFloat64())

	voidedType := customerbalance.CreditTransactionTypeVoided
	result, err := s.CustomerBalanceSvc.ListCreditTransactions(ctx, customerbalance.ListCreditTransactionsInput{
		CustomerID:    cust.GetID(),
		Limit:         20,
		Type:          &voidedType,
		FeatureFilter: customerbalance.NewFeatureFilter([]string{"api_requests_total"}),
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 1)
	s.Equal(float64(-100), result.Items[0].Amount.InexactFloat64())
	s.Equal(funding.Charge.ID, result.Items[0].Annotations[ledger.AnnotationChargeID])
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

func (s *VoidGrantTestSuite) TestListGrantStatusFiltersUseGrantLifecycleState() {
	// given:
	// - pending, active, expired, and voided grants created through service flows
	// when:
	// - grants are listed by public status
	// then:
	// - each filter returns only grants in the matching public lifecycle state
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("voidgrant-list-status")
	cust := s.setupVoidTestCustomer(ctx, ns)

	fundedAt := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	expiresAfter := datetime.MustParseDuration(s.T(), "PT1H")
	pendingEffectiveAt := fundedAt.Add(48 * time.Hour)

	activeGrant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "active grant",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(100),
		FundingMethod: creditgrant.FundingMethodNone,
	})
	s.Require().NoError(err)

	expiredGrant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "expired grant",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(200),
		FundingMethod: creditgrant.FundingMethodNone,
		ExpiresAfter:  &expiresAfter,
	})
	s.Require().NoError(err)

	voidedGrant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "voided grant",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(300),
		FundingMethod: creditgrant.FundingMethodNone,
	})
	s.Require().NoError(err)

	pendingGrant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "pending grant",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(400),
		EffectiveAt:   &pendingEffectiveAt,
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:         USD,
			PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
	})
	s.Require().NoError(err)

	voidedAt := fundedAt.Add(30 * time.Minute)
	clock.FreezeTime(voidedAt)

	_, err = s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ChargeID:   voidedGrant.ID,
	})
	s.Require().NoError(err)

	clock.FreezeTime(fundedAt.Add(2 * time.Hour))

	s.requireGrantIDsForStatusFilter(ctx, ns, cust.ID, creditgrant.GrantStatusPending, []string{pendingGrant.ID})
	s.requireGrantIDsForStatusFilter(ctx, ns, cust.ID, creditgrant.GrantStatusActive, []string{activeGrant.ID})
	s.requireGrantIDsForStatusFilter(ctx, ns, cust.ID, creditgrant.GrantStatusExpired, []string{expiredGrant.ID})
	s.requireGrantIDsForStatusFilter(ctx, ns, cust.ID, creditgrant.GrantStatusVoided, []string{voidedGrant.ID})
}

func (s *VoidGrantTestSuite) TestVoidInvoiceFundedGrantReceivableBalancesAcrossPaymentFlow() {
	// given:
	// - invoice-funded grants issued through the real invoice-backed credit-purchase flow
	// when:
	// - each grant is voided at a different payment lifecycle point
	// then:
	// - voiding corrects the original issuance, while already-booked payment auth/settle entries remain visible
	ctx := s.T().Context()
	issuedAt := datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime()
	amount := alpacadecimal.NewFromInt(100)
	costBasis := alpacadecimal.NewFromInt(1)

	tests := []struct {
		name           string
		namespace      string
		advance        func(context.Context, billing.StandardInvoice)
		wantOpen       alpacadecimal.Decimal
		wantAuthorized alpacadecimal.Decimal
		wantWash       alpacadecimal.Decimal
	}{
		{
			name:      "after issue before auth",
			namespace: "voidgrant-invoice-before-auth",
			advance: func(ctx context.Context, invoice billing.StandardInvoice) {
				clock.FreezeTime(issuedAt.Add(time.Hour))
				invoice, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
			},
			wantOpen:       alpacadecimal.Zero,
			wantAuthorized: alpacadecimal.Zero,
			wantWash:       alpacadecimal.Zero,
		},
		{
			name:      "after auth before settle",
			namespace: "voidgrant-invoice-after-auth",
			advance: func(ctx context.Context, invoice billing.StandardInvoice) {
				clock.FreezeTime(issuedAt.Add(time.Hour))
				invoice, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

				clock.FreezeTime(issuedAt.Add(2 * time.Hour))
				invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)
			},
			wantOpen:       amount,
			wantAuthorized: amount.Neg(),
			wantWash:       alpacadecimal.Zero,
		},
		{
			name:      "after settle",
			namespace: "voidgrant-invoice-after-settle",
			advance: func(ctx context.Context, invoice billing.StandardInvoice) {
				clock.FreezeTime(issuedAt.Add(time.Hour))
				invoice, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

				clock.FreezeTime(issuedAt.Add(2 * time.Hour))
				invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

				clock.FreezeTime(issuedAt.Add(3 * time.Hour))
				invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
					InvoiceID: invoice.GetInvoiceID(),
					Trigger:   billing.TriggerPaid,
				})
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
			},
			wantOpen:       amount,
			wantAuthorized: alpacadecimal.Zero,
			wantWash:       amount.Neg(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer clock.UnFreeze()

			ns := s.GetUniqueNamespace(tt.namespace)
			cust, grant, invoice := s.setupInvoiceFundedVoidGrant(ctx, ns, issuedAt, amount, costBasis)

			tt.advance(ctx, invoice)
			clock.FreezeTime(issuedAt.Add(4 * time.Hour))

			voidedGrant, err := s.CreditGrantService.Void(ctx, creditgrant.VoidInput{
				Namespace:  ns,
				CustomerID: cust.ID,
				ChargeID:   grant.ID,
			})
			s.Require().NoError(err)
			s.Require().NotNil(voidedGrant.State.VoidedAt)

			s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)), "void should remove the remaining grant value from FBO")
			s.AssertDecimalEqual(tt.wantOpen, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen), "open receivable balance after void")
			s.AssertDecimalEqual(tt.wantAuthorized, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized), "authorized receivable balance after void")
			s.AssertDecimalEqual(tt.wantWash, s.MustWashBalance(ns, USD, mo.Some(&costBasis)), "wash balance after void")
		})
	}
}

func (s *VoidGrantTestSuite) TestConcurrentVoidsDoNotDoubleBook() {
	// given:
	// - a funded grant of 100 voided concurrently from several goroutines
	// then:
	// - exactly one call books the void, the rest get a conflict, and FBO is cleared
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

	s.Equal(float64(0), s.MustBreakageBalanceAsOf(ns, USD, mo.None[*alpacadecimal.Decimal](), clock.Now()).InexactFloat64())
	s.Equal(float64(0), s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
	s.Equal(float64(100), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen).InexactFloat64())
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

func (s *VoidGrantTestSuite) setupInvoiceFundedVoidGrant(ctx context.Context, ns string, issuedAt time.Time, amount, costBasis alpacadecimal.Decimal) (*customer.Customer, creditpurchase.Charge, billing.StandardInvoice) {
	s.T().Helper()

	s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	clock.FreezeTime(issuedAt)
	defer clock.UnFreeze()

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "invoice-funded grant",
		Currency:      USD,
		Amount:        amount,
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:         USD,
			PerUnitCostBasis: &costBasis,
		},
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusActive, grant.Status)
	s.Require().NotNil(grant.Realizations.CreditGrantRealization)

	standardInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespaces: []string{ns},
		Expand:     billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(standardInvoices.Items, 1)
	invoice := standardInvoices.Items[0]
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(grant.ID, lo.FromPtr(invoice.Lines.OrEmpty()[0].ChargeID))

	s.AssertDecimalEqual(amount, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&costBasis)), "issued grant should be available in FBO before payment")
	s.AssertDecimalEqual(amount.Neg(), s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusOpen), "issued grant should create open receivable before payment")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.Some(&costBasis), ledger.TransactionAuthorizationStatusAuthorized), "issued grant should not be authorized yet")
	s.AssertDecimalEqual(alpacadecimal.Zero, s.MustWashBalance(ns, USD, mo.Some(&costBasis)), "issued grant should not be settled yet")

	return cust, grant, invoice
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
