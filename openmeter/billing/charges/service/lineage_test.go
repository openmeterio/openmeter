package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	lineage "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CreditRealizationLineageTestSuite struct {
	BaseSuite
}

func TestCreditRealizationLineage(t *testing.T) {
	suite.Run(t, new(CreditRealizationLineageTestSuite))
}

func (s *CreditRealizationLineageTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CreditRealizationLineageTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CreditRealizationLineageTestSuite) TestFlatFeeCreditOnlyAllocationCreatesInitialLineages() {
	defer s.FlatFeeTestHandler.Reset()

	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-realization-lineage")
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	s.FlatFeeTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        alpacadecimal.NewFromInt(20),
				Annotations:   creditrealization.LineageAnnotations(creditrealization.LineageOriginKindRealCredit),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        alpacadecimal.NewFromInt(30),
				Annotations:   creditrealization.LineageAnnotations(creditrealization.LineageOriginKindAdvance),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       currencyx.Code(currency.USD),
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(50),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:      "flat-fee-lineage",
				managedBy: billing.ManuallyManagedLine,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	chargeID, err := res[0].GetChargeID()
	s.NoError(err)

	charge, err := s.mustGetChargeByID(chargeID).AsFlatFeeCharge()
	s.NoError(err)
	s.Require().NotNil(charge.Realizations.CurrentRun)
	s.Len(charge.Realizations.CurrentRun.CreditRealizations, 2)

	lineages := s.mustListLineages(ns, realizationIDs(charge.Realizations.CurrentRun.CreditRealizations))
	s.Require().Len(lineages, 2)

	s.assertInitialLineage(lineages[charge.Realizations.CurrentRun.CreditRealizations[0].ID], chargeID.ID, charge.Realizations.CurrentRun.CreditRealizations[0].Amount, creditrealization.LineageOriginKindRealCredit, creditrealization.LineageSegmentStateRealCredit)
	s.assertInitialLineage(lineages[charge.Realizations.CurrentRun.CreditRealizations[1].ID], chargeID.ID, charge.Realizations.CurrentRun.CreditRealizations[1].Amount, creditrealization.LineageOriginKindAdvance, creditrealization.LineageSegmentStateAdvanceUncovered)
}

func (s *CreditRealizationLineageTestSuite) TestUsageBasedCreditOnlyAllocationCreatesInitialLineage() {
	defer s.UsageBasedTestHandler.Reset()

	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-usagebased-credit-realization-lineage")
	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	firstCollectionAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T12:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        input.AmountToAllocate,
				Annotations:   creditrealization.LineageAnnotations(creditrealization.LineageOriginKindAdvance),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       currencyx.Code(currency.USD),
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				name:       "usage-based-lineage",
				managedBy:  billing.ManuallyManagedLine,
				featureKey: meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	usageCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)

	clock.FreezeTime(firstCollectionAdvanceAt)
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		3,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
	s.Require().NotNil(advancedCharge)

	charge, err := s.mustGetChargeByID(usageCharge.GetChargeID()).AsUsageBasedCharge()
	s.NoError(err)
	s.Require().NotNil(charge.State.CurrentRealizationRunID)

	currentRun, err := charge.Realizations.GetByID(*charge.State.CurrentRealizationRunID)
	s.NoError(err)
	s.Len(currentRun.CreditsAllocated, 1)

	lineages := s.mustListLineages(ns, realizationIDs(currentRun.CreditsAllocated))
	s.Require().Len(lineages, 1)

	s.assertInitialLineage(lineages[currentRun.CreditsAllocated[0].ID], usageCharge.ID, currentRun.CreditsAllocated[0].Amount, creditrealization.LineageOriginKindAdvance, creditrealization.LineageSegmentStateAdvanceUncovered)
}

func (s *CreditRealizationLineageTestSuite) TestLockAdvanceLineagesForBackfillRequiresTransaction() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-lineage-lock-tx")
	adapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	_, err = adapter.LockAdvanceLineagesForBackfill(ctx, ns, "customer-id", currencyx.Code(currency.USD))
	s.Error(err)
	s.ErrorContains(err, "must be called in a transaction")
}

func (s *CreditRealizationLineageTestSuite) TestLockAdvanceLineagesForBackfillWorksInTransaction() {
	ctx, rawConfig, eDriver, err := s.DBClient.HijackTx(context.Background(), &sql.TxOptions{ReadOnly: false})
	s.Require().NoError(err)

	tx := entutils.NewTxDriver(eDriver, rawConfig)
	ctx, err = transaction.SetDriverOnContext(ctx, tx)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_ = tx.Rollback()
	})

	ns := s.GetUniqueNamespace("charges-service-lineage-lock-in-tx")
	adapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	lineages, err := adapter.LockAdvanceLineagesForBackfill(ctx, ns, "customer-id", currencyx.Code(currency.USD))
	s.NoError(err)
	s.Empty(lineages)
}

func (s *CreditRealizationLineageTestSuite) TestPersistCorrectionLineageSegmentsConsumesBackfilledBeforeUncovered() {
	ctx := context.Background()
	adapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	service, err := lineageservice.New(lineageservice.Config{
		Adapter: adapter,
	})
	s.Require().NoError(err)

	ns := s.GetUniqueNamespace("charges-service-lineage-correction-persist")
	backingTransactionGroupID := ulid.Make().String()
	lineageID := ulid.Make().String()
	chargeID := ulid.Make().String()
	rootRealizationID := ulid.Make().String()

	_, err = s.DBClient.Charge.Create().
		SetID(chargeID).
		SetNamespace(ns).
		SetType(chargesmeta.ChargeTypeFlatFee).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.DBClient.CreditRealizationLineage.Create().
		SetID(lineageID).
		SetNamespace(ns).
		SetChargeID(chargeID).
		SetRootRealizationID(rootRealizationID).
		SetCustomerID(ulid.Make().String()).
		SetCurrency(currencyx.Code(currency.USD)).
		SetOriginKind(creditrealization.LineageOriginKindAdvance).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.DBClient.CreditRealizationLineageSegment.CreateBulk(
		s.DBClient.CreditRealizationLineageSegment.Create().
			SetID(ulid.Make().String()).
			SetLineageID(lineageID).
			SetAmount(alpacadecimal.NewFromInt(20)).
			SetState(creditrealization.LineageSegmentStateAdvanceBackfilled).
			SetBackingTransactionGroupID(backingTransactionGroupID),
		s.DBClient.CreditRealizationLineageSegment.Create().
			SetID(ulid.Make().String()).
			SetLineageID(lineageID).
			SetAmount(alpacadecimal.NewFromInt(30)).
			SetState(creditrealization.LineageSegmentStateAdvanceUncovered),
	).Save(ctx)
	s.Require().NoError(err)

	err = service.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace: ns,
		Realizations: creditrealization.Realizations{
			{
				CreateInput: creditrealization.CreateInput{
					Type:                  creditrealization.TypeCorrection,
					Amount:                alpacadecimal.NewFromInt(-15),
					CorrectsRealizationID: lo.ToPtr(rootRealizationID),
				},
			},
		},
	})
	s.Require().NoError(err)

	activeSegments, err := s.DBClient.CreditRealizationLineageSegment.Query().
		Where(
			creditrealizationlineagesegment.LineageIDEQ(lineageID),
			creditrealizationlineagesegment.ClosedAtIsNil(),
		).
		Order(creditrealizationlineagesegment.ByCreatedAt()).
		All(ctx)
	s.Require().NoError(err)
	s.Require().Len(activeSegments, 2)

	s.Equal(creditrealization.LineageSegmentStateAdvanceUncovered, activeSegments[0].State)
	s.Equal(alpacadecimal.NewFromInt(30), activeSegments[0].Amount)
	s.Nil(activeSegments[0].BackingTransactionGroupID)

	s.Equal(creditrealization.LineageSegmentStateAdvanceBackfilled, activeSegments[1].State)
	s.Equal(alpacadecimal.NewFromInt(5), activeSegments[1].Amount)
	s.Equal(backingTransactionGroupID, lo.FromPtr(activeSegments[1].BackingTransactionGroupID))
}

func (s *CreditRealizationLineageTestSuite) TestCreateSegmentRejectsInvalidInput() {
	ctx := context.Background()
	adapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	err = adapter.CreateSegment(ctx, lineage.CreateSegmentInput{
		LineageID: ulid.Make().String(),
		Amount:    alpacadecimal.NewFromInt(10),
		State:     creditrealization.LineageSegmentStateAdvanceBackfilled,
	})
	s.Error(err)
	s.ErrorContains(err, "backing transaction group id is required")
}

func (s *CreditRealizationLineageTestSuite) mustAdvanceSingleUsageBasedCharge(ctx context.Context, customerID customer.CustomerID) *usagebased.Charge {
	s.T().Helper()

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.NoError(err)

	if len(advancedCharges) == 0 {
		return nil
	}

	s.Require().Len(advancedCharges, 1)
	charge, err := advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)

	return &charge
}

func (s *CreditRealizationLineageTestSuite) mustListLineages(namespace string, realizationIDs []string) map[string]*entdb.CreditRealizationLineage {
	s.T().Helper()

	lineages, err := s.DBClient.CreditRealizationLineage.Query().
		Where(
			creditrealizationlineage.Namespace(namespace),
			creditrealizationlineage.RootRealizationIDIn(realizationIDs...),
		).
		WithSegments().
		All(s.T().Context())
	s.NoError(err)

	out := make(map[string]*entdb.CreditRealizationLineage, len(lineages))
	for _, lineage := range lineages {
		out[lineage.RootRealizationID] = lineage
	}

	return out
}

func (s *CreditRealizationLineageTestSuite) assertInitialLineage(lineage *entdb.CreditRealizationLineage, chargeID string, amount alpacadecimal.Decimal, originKind creditrealization.LineageOriginKind, state creditrealization.LineageSegmentState) {
	s.T().Helper()

	require.NotNil(s.T(), lineage)
	s.Equal(chargeID, lineage.ChargeID)
	s.Equal(originKind, lineage.OriginKind)
	s.Require().Len(lineage.Edges.Segments, 1)
	s.Equal(amount, lineage.Edges.Segments[0].Amount)
	s.Equal(state, lineage.Edges.Segments[0].State)
	s.Nil(lineage.Edges.Segments[0].ClosedAt)
	s.Nil(lineage.Edges.Segments[0].BackingTransactionGroupID)
}

func realizationIDs(realizations creditrealization.Realizations) []string {
	return lo.Map(realizations, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})
}
