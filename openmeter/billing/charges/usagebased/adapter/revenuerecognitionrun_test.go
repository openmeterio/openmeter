package adapter

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	dbchargeusagebasedrevenuerecognitions "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrevenuerecognitions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *DetailedLineAdapterSuite) TestRevenueRecognitionRunCreateAndGetByID() {
	ctx := s.T().Context()
	namespace := "usagebased-revenue-recognition-run"

	// Given a usage-based charge.
	charge, _, servicePeriod := s.createChargeWithRun(namespace)
	dayBoundary := servicePeriod.From.AddDate(0, 0, 1)

	// When a revenue recognition run is created for the first day.
	created, err := s.adapter.CreateRevenueRecognitionRun(ctx, usagebased.CreateRevenueRecognitionRunInput{
		ChargeID:        charge.GetChargeID(),
		FeatureID:       charge.State.FeatureID,
		StoredAtLT:      dayBoundary.Add(time.Minute),
		ServicePeriodTo: dayBoundary,
		MeteredQuantity: alpacadecimal.NewFromInt(30),
		Totals: totals.DeltaTotals{
			Totals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(3),
				CreditsTotal: alpacadecimal.NewFromInt(1),
				Total:        alpacadecimal.NewFromInt(2),
			},
		},
		DetailedLinesIncludeCreditAllocations: true,
	})
	s.Require().NoError(err)

	// Then reading it by ID returns what was stored.
	fetched, err := s.adapter.GetRevenueRecognitionRunByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created, fetched)

	s.Equal(namespace, fetched.ID.Namespace)
	s.Equal(charge.ID, fetched.ChargeID)
	s.Equal(charge.State.FeatureID, fetched.FeatureID)
	s.Equal(dayBoundary.Add(time.Minute), fetched.StoredAtLT)
	s.Equal(dayBoundary, fetched.ServicePeriodTo)
	s.Nil(fetched.PriorRevRecogRunID)
	s.Equal(float64(30), fetched.MeteredQuantity.InexactFloat64())
	s.Equal(float64(3), fetched.Totals.Amount.InexactFloat64())
	s.Equal(float64(1), fetched.Totals.CreditsTotal.InexactFloat64())
	s.Equal(float64(2), fetched.Totals.Total.InexactFloat64())
	s.True(fetched.DetailedLinesIncludeCreditAllocations)
	s.Nil(fetched.DeletedAt)

	// And the run is not visible from another namespace or under an unknown ID.
	_, err = s.adapter.GetRevenueRecognitionRunByID(ctx, usagebased.RevenueRecognitionRunID{
		Namespace: namespace + "-other",
		ID:        created.ID.ID,
	})
	s.Require().Error(err)
	s.True(models.IsGenericNotFoundError(err))

	_, err = s.adapter.GetRevenueRecognitionRunByID(ctx, usagebased.RevenueRecognitionRunID{
		Namespace: namespace,
		ID:        ulid.Make().String(),
	})
	s.Require().Error(err)
	s.True(models.IsGenericNotFoundError(err))
}

func (s *DetailedLineAdapterSuite) TestRevenueRecognitionRunPointsAtPriorRun() {
	ctx := s.T().Context()
	namespace := "usagebased-revenue-recognition-run-prior"

	// Given a usage-based charge with a first revenue recognition run.
	charge, _, servicePeriod := s.createChargeWithRun(namespace)
	firstDay := servicePeriod.From.AddDate(0, 0, 1)

	input := usagebased.CreateRevenueRecognitionRunInput{
		ChargeID:        charge.GetChargeID(),
		FeatureID:       charge.State.FeatureID,
		StoredAtLT:      firstDay,
		ServicePeriodTo: firstDay,
	}

	firstRun, err := s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().NoError(err)

	// When the next day's run names the first run as its prior run.
	secondDay := firstDay.AddDate(0, 0, 1)
	input.StoredAtLT = secondDay
	input.ServicePeriodTo = secondDay
	input.PriorRevRecogRunID = &firstRun.ID

	secondRun, err := s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().NoError(err)

	// Then the pointer is stored and read back.
	fetched, err := s.adapter.GetRevenueRecognitionRunByID(ctx, secondRun.ID)
	s.Require().NoError(err)
	s.Require().NotNil(fetched.PriorRevRecogRunID)
	s.Equal(firstRun.ID, *fetched.PriorRevRecogRunID)

	// And a prior run from another namespace is rejected.
	input.PriorRevRecogRunID = &usagebased.RevenueRecognitionRunID{
		Namespace: namespace + "-other",
		ID:        firstRun.ID.ID,
	}
	_, err = s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().Error(err)
	s.True(models.IsGenericValidationError(err))
}

func (s *DetailedLineAdapterSuite) TestRevenueRecognitionRunCreateRollsBackWhenChildInsertFails() {
	ctx := s.T().Context()
	namespace := "usagebased-revenue-recognition-run-rollback"

	// Given a usage-based charge.
	charge, _, servicePeriod := s.createChargeWithRun(namespace)
	dayPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.AddDate(0, 0, 1),
	}
	duplicateID := ulid.Make().String()
	ledgerTransaction := ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}

	// When a run is created with two credits that share an ID, so the credit
	// insert fails after the run row was written.
	_, err := s.adapter.CreateRevenueRecognitionRun(ctx, usagebased.CreateRevenueRecognitionRunInput{
		ChargeID:        charge.GetChargeID(),
		FeatureID:       charge.State.FeatureID,
		StoredAtLT:      dayPeriod.To,
		ServicePeriodTo: dayPeriod.To,
		CreditsAllocated: creditrealization.CreateInputs{
			{
				ID:                duplicateID,
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(1),
				Type:              creditrealization.TypeAllocation,
			},
			{
				ID:                duplicateID,
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(2),
				Type:              creditrealization.TypeAllocation,
			},
		},
	})
	s.Require().Error(err)
	s.False(models.IsGenericValidationError(err))

	// Then the run is not stored either.
	runCount, err := s.dbClient.ChargeUsageBasedRevenueRecognitions.Query().
		Where(dbchargeusagebasedrevenuerecognitions.Namespace(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, runCount)
}

func (s *DetailedLineAdapterSuite) TestRevenueRecognitionRunStoresCreditsInvoicedUsagesAndDetailedLines() {
	ctx := s.T().Context()
	namespace := "usagebased-revenue-recognition-run-children"

	// Given a usage-based charge.
	charge, _, servicePeriod := s.createChargeWithRun(namespace)
	dayPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.AddDate(0, 0, 1),
	}
	ledgerTransaction := ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}
	lineID := ulid.Make().String()

	// The first credit gets the larger ID, so reading back in input order shows
	// that the sort hint decides the order, not the ID.
	smallerID, largerID := ulid.Make().String(), ulid.Make().String()
	if smallerID > largerID {
		smallerID, largerID = largerID, smallerID
	}

	input := usagebased.CreateRevenueRecognitionRunInput{
		ChargeID:        charge.GetChargeID(),
		FeatureID:       charge.State.FeatureID,
		StoredAtLT:      dayPeriod.To,
		ServicePeriodTo: dayPeriod.To,
		MeteredQuantity: alpacadecimal.NewFromInt(3),
		Totals: totals.DeltaTotals{
			Totals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(4),
				CreditsTotal: alpacadecimal.NewFromInt(2),
				Total:        alpacadecimal.NewFromInt(2),
			},
		},
		CreditsAllocated: creditrealization.CreateInputs{
			{
				ID:                largerID,
				Annotations:       models.Annotations{"source": "test"},
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(3),
				LineID:            lo.ToPtr(lineID),
				Type:              creditrealization.TypeAllocation,
			},
			{
				ID:                smallerID,
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(-1),
				Type:              creditrealization.TypeCorrection,
			},
		},
		InvoicedUsages: creditrealization.CreateInputs{
			{
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(5),
				Type:              creditrealization.TypeAllocation,
			},
			{
				ServicePeriod:     dayPeriod,
				LedgerTransaction: ledgerTransaction,
				Amount:            alpacadecimal.NewFromInt(-3),
				Type:              creditrealization.TypeCorrection,
			},
		},
		DetailedLines: []usagebased.RevenueRecognitionDetailedLine{
			{
				ManagedResource: models.ManagedResource{
					Name:        "Second line",
					Description: lo.ToPtr("stored after the first line"),
				},
				Currency:               "USD",
				ServicePeriod:          dayPeriod,
				ChildUniqueReferenceID: "unit-price-usage-tier-2",
				Index:                  lo.ToPtr(1),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          alpacadecimal.NewFromInt(1),
				Amount:                 alpacadecimal.NewFromInt(1),
			},
			{
				ManagedResource: models.ManagedResource{
					Name: "First line",
				},
				Annotations:            models.Annotations{"source": "test"},
				Metadata:               models.Metadata{"tier": "1"},
				Currency:               "USD",
				ServicePeriod:          dayPeriod,
				ChildUniqueReferenceID: "unit-price-usage-tier-1",
				Index:                  lo.ToPtr(0),
				Quantity:               alpacadecimal.NewFromInt(2),
				PerUnitAmount:          alpacadecimal.NewFromFloat(1.5),
				Amount:                 alpacadecimal.NewFromInt(3),
				AmountDiscounts: chargedetailedline.AmountDiscounts{
					{
						Amount:                 alpacadecimal.NewFromInt(1),
						ChildUniqueReferenceID: "usage-discount",
						Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					},
				},
				CreditsApplied: creditsapplied.CreditsApplied{
					{
						Amount:      alpacadecimal.NewFromInt(2),
						Description: "Credits",
					},
				},
			},
		},
	}

	// When the run is created with credits, invoiced usages and detailed lines.
	created, err := s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().NoError(err)

	// Then reading it by ID returns all of them.
	fetched, err := s.adapter.GetRevenueRecognitionRunByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created, fetched)

	// Credits come back in input order through the sort hint.
	s.Require().Len(fetched.CreditsAllocated, 2)
	allocation, correction := fetched.CreditsAllocated[0], fetched.CreditsAllocated[1]

	s.Equal(largerID, allocation.ID)
	s.Equal(0, allocation.SortHint)
	s.Equal(creditrealization.TypeAllocation, allocation.Type)
	s.Equal(namespace, allocation.Namespace)
	s.Equal(models.Annotations{"source": "test"}, allocation.Annotations)
	s.Equal(dayPeriod, allocation.ServicePeriod)
	s.Equal(float64(3), allocation.Amount.InexactFloat64())
	s.Equal(ledgerTransaction, allocation.LedgerTransaction)
	s.Equal(lo.ToPtr(lineID), allocation.LineID)

	s.Equal(smallerID, correction.ID)
	s.Equal(1, correction.SortHint)
	s.Equal(creditrealization.TypeCorrection, correction.Type)
	s.Equal(float64(-1), correction.Amount.InexactFloat64())
	s.Nil(correction.LineID)
	s.Nil(correction.CorrectsRealizationID)

	// Invoiced usages are separate rows, and their amounts add up to the run's
	// invoiced total.
	s.Require().Len(fetched.InvoicedUsages, 2)
	s.Equal(creditrealization.TypeAllocation, fetched.InvoicedUsages[0].Type)
	s.Equal(float64(5), fetched.InvoicedUsages[0].Amount.InexactFloat64())
	s.Equal(ledgerTransaction, fetched.InvoicedUsages[0].LedgerTransaction)
	s.Equal(creditrealization.TypeCorrection, fetched.InvoicedUsages[1].Type)
	s.Equal(float64(-3), fetched.InvoicedUsages[1].Amount.InexactFloat64())
	s.Equal(1, fetched.InvoicedUsages[1].SortHint)

	// Detailed lines come back in index order, not insertion order.
	s.Require().Len(fetched.DetailedLines, 2)
	firstLine, secondLine := fetched.DetailedLines[0], fetched.DetailedLines[1]

	s.NotEmpty(firstLine.ID)
	s.Equal(namespace, firstLine.Namespace)
	s.Equal("First line", firstLine.Name)
	s.Nil(firstLine.Description)
	s.Equal(models.Annotations{"source": "test"}, firstLine.Annotations)
	s.Equal(models.Metadata{"tier": "1"}, firstLine.Metadata)
	s.Equal(currencyx.Code("USD"), firstLine.Currency)
	s.Equal(dayPeriod, firstLine.ServicePeriod)
	s.Equal("unit-price-usage-tier-1", firstLine.ChildUniqueReferenceID)
	s.Equal(lo.ToPtr(0), firstLine.Index)
	s.Equal(float64(2), firstLine.Quantity.InexactFloat64())
	s.Equal(float64(1.5), firstLine.PerUnitAmount.InexactFloat64())
	s.Equal(float64(3), firstLine.Amount.InexactFloat64())
	s.Require().Len(firstLine.AmountDiscounts, 1)
	s.Equal("usage-discount", firstLine.AmountDiscounts[0].ChildUniqueReferenceID)
	s.Equal(float64(1), firstLine.AmountDiscounts[0].Amount.InexactFloat64())
	s.Require().Len(firstLine.CreditsApplied, 1)
	s.Equal(float64(2), firstLine.CreditsApplied[0].Amount.InexactFloat64())

	s.Equal("Second line", secondLine.Name)
	s.Equal(lo.ToPtr("stored after the first line"), secondLine.Description)
	s.Equal("unit-price-usage-tier-2", secondLine.ChildUniqueReferenceID)
	s.Empty(secondLine.AmountDiscounts)
	s.Empty(secondLine.CreditsApplied)

	// And a correction that names the realization it corrects is rejected, because
	// there is no column to store that in.
	input.CreditsAllocated[0].ID, input.CreditsAllocated[1].ID = "", ""
	input.CreditsAllocated[1].CorrectsRealizationID = lo.ToPtr(allocation.ID)
	_, err = s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().Error(err)
	s.True(models.IsGenericValidationError(err))

	// And so is an invoiced usage without a ledger transaction.
	input.CreditsAllocated[1].CorrectsRealizationID = nil
	input.InvoicedUsages[0].LedgerTransaction = ledgertransaction.GroupReference{}
	_, err = s.adapter.CreateRevenueRecognitionRun(ctx, input)
	s.Require().Error(err)
	s.True(models.IsGenericValidationError(err))
}
