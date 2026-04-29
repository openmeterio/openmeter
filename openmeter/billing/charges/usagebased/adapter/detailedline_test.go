package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebasedrundetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrundetailedline"
	dbchargeusagebasedruns "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruns"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestDetailedLineAdapter(t *testing.T) {
	suite.Run(t, new(DetailedLineAdapterSuite))
}

type DetailedLineAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  usagebased.Adapter
}

type newDetailedLineInput struct {
	Charge                 usagebased.Charge
	RunID                  usagebased.RealizationRunID
	ServicePeriod          timeutil.ClosedPeriod
	ChildUniqueReferenceID string
	Quantity               int64
	Description            *string
}

func (s *DetailedLineAdapterSuite) SetupSuite() {
	t := s.T()

	s.testDB = testutils.InitPostgresDB(t)
	s.dbClient = entdb.NewClient(entdb.Driver(s.testDB.EntDriver.Driver()))

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: s.testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           slog.Default(),
	})
	require.NoError(t, err)
	defer migrator.CloseOrLogError()
	require.NoError(t, migrator.Up())

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	a, err := New(Config{
		Client:      s.dbClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	require.NoError(t, err)

	s.adapter = a
}

func (s *DetailedLineAdapterSuite) TearDownSuite() {
	s.dbClient.Close()
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *DetailedLineAdapterSuite) TestUpsertRunDetailedLinesReplacesAndSoftDeletesByChildUniqueReferenceID() {
	ctx := s.T().Context()
	namespace := "usagebased-detailedline-adapter"
	customerID := s.createCustomer(namespace)
	s.createFeature(namespace, "feature-1")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(ctx, usagebased.CreateChargesInput{
		Namespace: namespace,
		Intents: []usagebased.CreateIntent{
			{
				Intent: usagebased.Intent{
					Intent: chargesmeta.Intent{
						Name:              "usage-charge",
						ManagedBy:         billing.SubscriptionManagedLine,
						UniqueReferenceID: nil,
						CustomerID:        customerID,
						Currency:          currencyx.Code("USD"),
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:      servicePeriod.To,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					FeatureKey:     "feature-1",
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(0.1),
					}),
				},
				FeatureID: "feature-1",
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	charge := createdCharges[0]
	runBase, err := s.adapter.CreateRealizationRun(ctx, charge.GetChargeID(), usagebased.CreateRealizationRunInput{
		FeatureID:       "feature-1",
		Type:            usagebased.RealizationRunTypeFinalRealization,
		StoredAtLT:      servicePeriod.To,
		ServicePeriodTo: servicePeriod.To,
		MeteredQuantity: alpacadecimal.NewFromInt(10),
		Totals: totals.Totals{
			Amount:       alpacadecimal.NewFromInt(1),
			ChargesTotal: alpacadecimal.NewFromInt(1),
			Total:        alpacadecimal.NewFromInt(1),
		},
	})
	s.Require().NoError(err)

	initialLines := usagebased.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               1,
			Description:            lo.ToPtr("old description"),
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "delete@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               2,
			Description:            lo.ToPtr("delete me"),
		}),
	}
	s.Require().NoError(s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), runBase.ID, initialLines))

	replacementLines := usagebased.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               3,
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "new@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               4,
			Description:            lo.ToPtr("new description"),
		}),
	}
	s.Require().NoError(s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), runBase.ID, replacementLines))

	fetchedCharge, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedCharge.Realizations, 1)
	s.True(fetchedCharge.Realizations[0].DetailedLines.IsPresent())
	s.Len(fetchedCharge.Realizations[0].DetailedLines.OrEmpty(), 2)
	s.Equal("keep@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]", fetchedCharge.Realizations[0].DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
	s.Equal("new@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]", fetchedCharge.Realizations[0].DetailedLines.OrEmpty()[1].ChildUniqueReferenceID)
	s.Equal(float64(3), fetchedCharge.Realizations[0].DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
	s.Nil(fetchedCharge.Realizations[0].DetailedLines.OrEmpty()[0].Description)

	deletedRow, err := s.dbClient.ChargeUsageBasedRunDetailedLine.Query().
		Where(
			dbchargeusagebasedrundetailedline.NamespaceEQ(namespace),
			dbchargeusagebasedrundetailedline.ChargeIDEQ(charge.ID),
			dbchargeusagebasedrundetailedline.RunIDEQ(runBase.ID.ID),
			dbchargeusagebasedrundetailedline.ChildUniqueReferenceIDEQ("delete@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]"),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.NotNil(deletedRow.DeletedAt)
}

func (s *DetailedLineAdapterSuite) TestFetchDetailedLinesUsesDetailedLinesPresentFlag() {
	ctx := s.T().Context()
	namespace := "usagebased-detailedline-adapter-fetch-flag"
	charge, runBase, _ := s.createChargeWithRun(namespace)

	fetchedWithoutMaterializedLines, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedWithoutMaterializedLines.Realizations, 1)
	s.False(fetchedWithoutMaterializedLines.Realizations[0].DetailedLines.IsPresent())

	s.Require().NoError(s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), runBase.ID, nil))

	fetchedWithMaterializedEmptyLines, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedWithMaterializedEmptyLines.Realizations, 1)
	s.True(fetchedWithMaterializedEmptyLines.Realizations[0].DetailedLines.IsPresent())
	s.Empty(fetchedWithMaterializedEmptyLines.Realizations[0].DetailedLines.OrEmpty())

	dbRun, err := s.dbClient.ChargeUsageBasedRuns.Query().
		Where(
			dbchargeusagebasedruns.NamespaceEQ(namespace),
			dbchargeusagebasedruns.ID(runBase.ID.ID),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.True(dbRun.DetailedLinesPresent)
}

func (s *DetailedLineAdapterSuite) TestFetchDetailedLinesDoesNotRepairDetailedLinesPresentFlagWhenRowsExist() {
	ctx := s.T().Context()
	namespace := "usagebased-detailedline-adapter-fetch-does-not-repair-flag"
	charge, runBase, servicePeriod := s.createChargeWithRun(namespace)

	s.Require().NoError(s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), runBase.ID, usagebased.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "existing@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               1,
		}),
	}))

	_, err := s.dbClient.ChargeUsageBasedRuns.UpdateOneID(runBase.ID.ID).
		Where(dbchargeusagebasedruns.NamespaceEQ(namespace)).
		SetDetailedLinesPresent(false).
		Save(ctx)
	s.Require().NoError(err)

	fetchedCharge, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedCharge.Realizations, 1)
	s.False(fetchedCharge.Realizations[0].DetailedLines.IsPresent())

	dbRun, err := s.dbClient.ChargeUsageBasedRuns.Query().
		Where(
			dbchargeusagebasedruns.NamespaceEQ(namespace),
			dbchargeusagebasedruns.ID(runBase.ID.ID),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.False(dbRun.DetailedLinesPresent)
}

func (s *DetailedLineAdapterSuite) TestFetchDetailedLinesUsesPersistedDetailedLinesPresentFlag() {
	ctx := s.T().Context()
	namespace := "usagebased-detailedline-adapter-fetch-uses-persisted-flag"
	charge, runBase, servicePeriod := s.createChargeWithRun(namespace)

	s.Require().NoError(s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), runBase.ID, usagebased.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "persisted@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               1,
		}),
	}))

	_, err := s.dbClient.ChargeUsageBasedRuns.UpdateOneID(runBase.ID.ID).
		Where(dbchargeusagebasedruns.NamespaceEQ(namespace)).
		SetDetailedLinesPresent(false).
		Save(ctx)
	s.Require().NoError(err)

	staleCharge := charge
	staleCharge.Realizations = usagebased.RealizationRuns{
		{
			RealizationRunBase: runBase,
		},
	}
	staleCharge.Realizations[0].DetailedLines = mo.Some(usagebased.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			RunID:                  runBase.ID,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "stale@[2026-01-01T00:00:00Z..2026-02-01T00:00:00Z]",
			Quantity:               1,
		}),
	})

	fetchedCharge, err := s.adapter.FetchDetailedLines(ctx, staleCharge)
	s.Require().NoError(err)
	s.Require().Len(fetchedCharge.Realizations, 1)
	s.False(fetchedCharge.Realizations[0].DetailedLines.IsPresent())
}

func (s *DetailedLineAdapterSuite) createChargeWithRun(namespace string) (usagebased.Charge, usagebased.RealizationRunBase, timeutil.ClosedPeriod) {
	s.T().Helper()

	featureID := ulid.Make().String()
	customerID := s.createCustomer(namespace)
	s.createFeature(namespace, featureID)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(s.T().Context(), usagebased.CreateChargesInput{
		Namespace: namespace,
		Intents: []usagebased.CreateIntent{
			{
				Intent: usagebased.Intent{
					Intent: chargesmeta.Intent{
						Name:              "usage-charge",
						ManagedBy:         billing.SubscriptionManagedLine,
						UniqueReferenceID: nil,
						CustomerID:        customerID,
						Currency:          currencyx.Code("USD"),
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:      servicePeriod.To,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					FeatureKey:     featureID,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(0.1),
					}),
				},
				FeatureID: featureID,
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	charge := createdCharges[0]
	runBase, err := s.adapter.CreateRealizationRun(s.T().Context(), charge.GetChargeID(), usagebased.CreateRealizationRunInput{
		FeatureID:       featureID,
		Type:            usagebased.RealizationRunTypeFinalRealization,
		StoredAtLT:      servicePeriod.To,
		ServicePeriodTo: servicePeriod.To,
		MeteredQuantity: alpacadecimal.NewFromInt(10),
		Totals: totals.Totals{
			Amount:       alpacadecimal.NewFromInt(1),
			ChargesTotal: alpacadecimal.NewFromInt(1),
			Total:        alpacadecimal.NewFromInt(1),
		},
	})
	s.Require().NoError(err)

	return charge, runBase, servicePeriod
}

func (s *DetailedLineAdapterSuite) createCustomer(namespace string) string {
	s.T().Helper()

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return customer.ID
}

func (s *DetailedLineAdapterSuite) createFeature(namespace, featureID string) {
	s.T().Helper()

	_, err := s.dbClient.Feature.Create().
		SetNamespace(namespace).
		SetID(featureID).
		SetName("test-feature").
		SetKey(featureID).
		Save(s.T().Context())
	s.Require().NoError(err)
}

func (s *DetailedLineAdapterSuite) newDetailedLine(input newDetailedLineInput) usagebased.DetailedLine {
	s.T().Helper()

	return usagebased.DetailedLine{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace:   input.Charge.Namespace,
			Name:        "Detailed line",
			Description: input.Description,
		}),
		ServicePeriod:          input.ServicePeriod,
		Currency:               input.Charge.Intent.Currency,
		ChildUniqueReferenceID: input.ChildUniqueReferenceID,
		PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		PerUnitAmount:          alpacadecimal.NewFromFloat(0.1),
		Quantity:               alpacadecimal.NewFromInt(input.Quantity),
		Category:               stddetailedline.CategoryRegular,
		Totals: totals.Totals{
			Amount:       alpacadecimal.NewFromFloat(0.1).Mul(alpacadecimal.NewFromInt(input.Quantity)),
			ChargesTotal: alpacadecimal.NewFromFloat(0.1).Mul(alpacadecimal.NewFromInt(input.Quantity)),
			Total:        alpacadecimal.NewFromFloat(0.1).Mul(alpacadecimal.NewFromInt(input.Quantity)),
		},
	}
}
