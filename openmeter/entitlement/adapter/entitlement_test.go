package adapter_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	featureadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var m sync.Mutex

type deps struct {
	entRepo      entitlement.EntitlementRepo
	featureRepo  feature.FeatureRepo
	subjectRepo  subject.Service
	customerRepo customer.Adapter
	meterRepo    *meteradapter.Adapter
}

func setup(t *testing.T) (deps deps, cleanup func()) {
	t.Helper()

	// create isolated pg db for tests
	testdb := testutils.InitPostgresDB(t)
	logger := testutils.NewLogger(t)

	logger.Info("testdb.URL", "testdb.URL", testdb.URL)

	dbClient := testdb.EntDriver.Client().Debug()
	pgDriver := testdb.PGDriver
	entDriver := testdb.EntDriver

	cleanup = func() {
		dbClient.Close()
		entDriver.Close()
		pgDriver.Close()
	}

	var err error

	// Init meter service
	deps.meterRepo, err = meteradapter.New(meteradapter.Config{
		Client: dbClient,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing meter adapter must not fail")
	require.NotNilf(t, deps.meterRepo, "meter adapter must not be nil")

	deps.entRepo = adapter.NewPostgresEntitlementRepo(dbClient)
	deps.featureRepo = featureadapter.NewPostgresFeatureRepo(dbClient, logger)

	// customer adapter for creating customers in tests
	custAdapter, err := customeradapter.New(customeradapter.Config{Client: dbClient, Logger: logger})
	if err != nil {
		t.Fatalf("failed to create customer adapter: %v", err)
	}
	deps.customerRepo = custAdapter

	// Create subject adapter and service
	subjectAdapter, err := subjectadapter.New(dbClient)
	if err != nil {
		t.Fatalf("failed to create subject adapter: %v", err)
	}

	subjectService, err := subjectservice.New(subjectAdapter)
	if err != nil {
		t.Fatalf("failed to create subject service: %v", err)
	}
	deps.subjectRepo = subjectService

	m.Lock()
	defer m.Unlock()
	// migrate db via ent schema upsert
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return deps, cleanup
}

func createCustomerWithSubject(t *testing.T, subjectRepo subject.Service, customerRepo customer.Adapter, namespace string, subjectKey string) *customer.Customer {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := subjectRepo.Create(ctx, subject.CreateInput{
		Namespace: namespace,
		Key:       subjectKey,
	})

	require.NoError(t, err)

	cust, err := customerRepo.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Customer 1",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{subjectKey},
			},
		},
	})

	require.NoError(t, err)

	return cust
}

func createCustomerWithSubjectAndKey(t *testing.T, subjectRepo subject.Service, customerRepo customer.Adapter, namespace string, subjectKey string, customerKey string) *customer.Customer {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := subjectRepo.Create(ctx, subject.CreateInput{
		Namespace: namespace,
		Key:       subjectKey,
	})
	require.NoError(t, err)

	cust, err := customerRepo.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(customerKey),
			Name: "Customer 1",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{subjectKey},
			},
		},
	})
	require.NoError(t, err)

	return cust
}

func TestUpsertEntitlementCurrentPeriods(t *testing.T) {
	ns := "ns1"
	featureKey := "feature1"

	t.Run("Should upsert entitlement current periods but no other fields", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		// First, let's create the subjects
		cust1 := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subject1")
		cust2 := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subject2")
		cust3 := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subject3")

		// Then, let's create 3 entitlements
		ent1, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			UsageAttribution: cust1.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})),
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent1)

		ent2, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			UsageAttribution: cust2.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})),
			CurrentUsagePeriod: nil, // notice we don't have a current period set here
		})
		require.NoError(t, err)
		require.NotNil(t, ent2)

		ent3, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			UsageAttribution: cust3.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})),
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent3)

		ent1NewPeriod := timeutil.ClosedPeriod{
			From: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		}

		ent2NewPeriod := timeutil.ClosedPeriod{
			From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		}

		// Then let's upsert two of them
		err = repo.entRepo.UpsertEntitlementCurrentPeriods(ctx, []entitlement.UpsertEntitlementCurrentPeriodElement{
			{
				NamespacedID: models.NamespacedID{
					ID:        ent1.ID,
					Namespace: ent1.Namespace,
				},
				CurrentUsagePeriod: ent1NewPeriod,
			},
			{
				NamespacedID: models.NamespacedID{
					ID:        ent2.ID,
					Namespace: ent2.Namespace,
				},
				CurrentUsagePeriod: ent2NewPeriod,
			},
		})
		require.NoError(t, err)

		// Let's check we still only have 3 entitlements
		entitlements, err := repo.entRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{})
		require.NoError(t, err)
		require.Equal(t, 3, len(entitlements.Items))

		// To avoid mapping to calculated values, we need to use ListActiveEntitlementsWithExpiredUsagePeriod
		ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC),
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(ents))

		// Let's check that their current periods are updated and no other fields are touched
		ent1Updated, ok := lo.Find(ents, func(e entitlement.Entitlement) bool {
			return e.ID == ent1.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent1Updated)
		require.Equal(t, ent1.ID, ent1Updated.ID)
		require.NotNil(t, ent1Updated.CurrentUsagePeriod)
		require.Equal(t, ent1NewPeriod, *ent1Updated.CurrentUsagePeriod)
		require.Equal(t, ent1.FeatureID, ent1Updated.FeatureID)
		require.Equal(t, ent1.FeatureKey, ent1Updated.FeatureKey)

		require.Equal(t, ent1.EntitlementType, ent1Updated.EntitlementType)
		require.Equal(t, ent1.MeasureUsageFrom.UTC(), ent1Updated.MeasureUsageFrom.UTC())
		require.True(t, ent1.UsagePeriod.Equal(*ent1Updated.UsagePeriod), "usage period should be equal, got %+v and %+v", ent1.UsagePeriod, ent1Updated.UsagePeriod)

		ent2Updated, ok := lo.Find(ents, func(e entitlement.Entitlement) bool {
			return e.ID == ent2.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent2Updated)
		require.Equal(t, ent2.ID, ent2Updated.ID)
		require.NotNil(t, ent2Updated.CurrentUsagePeriod)
		require.Equal(t, ent2NewPeriod, *ent2Updated.CurrentUsagePeriod)
		require.Equal(t, ent2.FeatureID, ent2Updated.FeatureID)
		require.Equal(t, ent2.FeatureKey, ent2Updated.FeatureKey)
		require.Equal(t, ent2.EntitlementType, ent2Updated.EntitlementType)
		require.Equal(t, ent2.MeasureUsageFrom.UTC(), ent2Updated.MeasureUsageFrom.UTC())
		require.True(t, ent2.UsagePeriod.Equal(*ent2Updated.UsagePeriod), "usage period should be equal, got %+v and %+v", ent2.UsagePeriod, ent2Updated.UsagePeriod)

		// Let's check that the other one is not touched
		ent3Updated, ok := lo.Find(entitlements.Items, func(e entitlement.Entitlement) bool {
			return e.ID == ent3.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent3Updated)
		require.Equal(t, ent3.ID, ent3Updated.ID)
		require.Equal(t, ent3.CurrentUsagePeriod, ent3Updated.CurrentUsagePeriod)
		require.Equal(t, ent3.FeatureID, ent3Updated.FeatureID)
		require.Equal(t, ent3.FeatureKey, ent3Updated.FeatureKey)
		require.Equal(t, ent3.EntitlementType, ent3Updated.EntitlementType)
		require.Equal(t, ent3.MeasureUsageFrom.UTC(), ent3Updated.MeasureUsageFrom.UTC())
		require.True(t, ent3.UsagePeriod.Equal(*ent3Updated.UsagePeriod), "usage period should be equal, got %+v and %+v", ent3.UsagePeriod, ent3Updated.UsagePeriod)
	})
}

func TestListActiveEntitlementsWithExpiredUsagePeriod(t *testing.T) {
	ns := "ns1"
	featureKey := "feature1"

	t.Run("Should return entitlements with expired usage period", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		// Let's set the current time
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		// First, create the subjects
		cust1 := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subject1")
		cust2 := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subject2")

		// Then create two entitlements, one with expired usage period and one with no expired usage period
		ent1, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			UsageAttribution: cust1.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})),
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent1)

		ent2, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			UsageAttribution: cust2.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			})),
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent2)

		now = time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's check that the entitlement with expired usage period is returned
		ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: now,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(ents))
		require.Equal(t, ent1.ID, ents[0].ID)
	})

	t.Run("Should return entitlements with cursor and limit", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		now := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		var ents []entitlement.Entitlement

		// Let's create 10 entitlements (with their subjects first)
		for i := 0; i < 10; i++ {
			subjectKey := fmt.Sprintf("subject%d", i)
			cust := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, subjectKey)

			ent, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
				Namespace:        ns,
				FeatureID:        feature.ID,
				FeatureKey:       featureKey,
				UsageAttribution: cust.GetUsageAttribution(),
				EntitlementType:  entitlement.EntitlementTypeMetered,
				MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				})),
				CurrentUsagePeriod: &timeutil.ClosedPeriod{
					From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			})
			time.Sleep(1 * time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, ent)
			ents = append(ents, *ent)
		}

		// Let's check that the entitlements are returned
		resetableEnts, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			Limit:         5,
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(resetableEnts))
		require.Equal(t, ents[0].ID, resetableEnts[0].ID)
		require.Equal(t, ents[1].ID, resetableEnts[1].ID)
		require.Equal(t, ents[2].ID, resetableEnts[2].ID)
		require.Equal(t, ents[3].ID, resetableEnts[3].ID)
		require.Equal(t, ents[4].ID, resetableEnts[4].ID)

		// Let's query the next 5 entitlements
		next5Ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			Cursor:        lo.ToPtr(pagination.NewCursor(resetableEnts[4].CreatedAt, resetableEnts[4].ID)),
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(next5Ents))
		require.Equal(t, ents[5].ID, next5Ents[0].ID)
		require.Equal(t, ents[6].ID, next5Ents[1].ID)
		require.Equal(t, ents[7].ID, next5Ents[2].ID)
		require.Equal(t, ents[8].ID, next5Ents[3].ID)
		require.Equal(t, ents[9].ID, next5Ents[4].ID)
	})
}

func TestEntitlementLoadsSubjectAndCustomerAndPreservesAcrossTypedMapping(t *testing.T) {
	ctx := context.Background()
	ns := "ns-load"
	featureKey := "feat-load"

	repo, cleanup := setup(t)
	defer cleanup()

	// Create feature
	feat, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: ns,
		Key:       featureKey,
		Name:      "Feature Load",
	})
	require.NoError(t, err)

	// Create subject
	cust := createCustomerWithSubject(t, repo.subjectRepo, repo.customerRepo, ns, "subj-load")

	// Create 3 entitlements of different types
	// metered
	entMetered, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        feat.ID,
		FeatureKey:       featureKey,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		})),
		IsSoftLimit: lo.ToPtr(true),
	})
	require.NoError(t, err)

	// static
	entStatic, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        feat.ID,
		FeatureKey:       featureKey,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeStatic,
		Config:           lo.ToPtr(`{"on":true}`),
	})
	require.NoError(t, err)

	// boolean
	entBoolean, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        feat.ID,
		FeatureKey:       featureKey,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	})
	require.NoError(t, err)

	// Fetch individually and assert CustomerID is populated
	for _, id := range []string{entMetered.ID, entStatic.ID, entBoolean.ID} {
		got, err := repo.entRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: ns, ID: id})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotEmpty(t, got.CustomerID)

		// Verify preservation through typed mapping
		switch got.EntitlementType {
		case entitlement.EntitlementTypeMetered:
			typed, err := meteredentitlement.ParseFromGenericEntitlement(got)
			require.NoError(t, err)
			require.NotEmpty(t, typed.CustomerID)
		case entitlement.EntitlementTypeStatic:
			typed, err := staticentitlement.ParseFromGenericEntitlement(got)
			require.NoError(t, err)
			require.NotEmpty(t, typed.CustomerID)
		case entitlement.EntitlementTypeBoolean:
			typed, err := booleanentitlement.ParseFromGenericEntitlement(got)
			require.NoError(t, err)
			require.NotEmpty(t, typed.CustomerID)
		}
	}
}

func TestListEntitlementsFiltersByCustomerKeysAndFeatureIDsOrKeys(t *testing.T) {
	ctx := context.Background()
	ns := "ns-list-filters"

	repo, cleanup := setup(t)
	defer cleanup()

	featureByKey, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: ns,
		Key:       "free_plan_usage",
		Name:      "Free plan usage",
	})
	require.NoError(t, err)

	featureByID, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: ns,
		Key:       "pro_plan_usage",
		Name:      "Pro plan usage",
	})
	require.NoError(t, err)

	customerA := createCustomerWithSubjectAndKey(t, repo.subjectRepo, repo.customerRepo, ns, "subject-a", "customer-a")
	customerB := createCustomerWithSubjectAndKey(t, repo.subjectRepo, repo.customerRepo, ns, "subject-b", "customer-b")

	entA, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        featureByKey.ID,
		FeatureKey:       featureByKey.Key,
		UsageAttribution: customerA.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	})
	require.NoError(t, err)

	entB, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        featureByID.ID,
		FeatureKey:       featureByID.Key,
		UsageAttribution: customerB.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	})
	require.NoError(t, err)

	t.Run("Should filter by customer key and feature key", func(t *testing.T) {
		res, err := repo.entRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
			Namespaces:       []string{ns},
			CustomerKeys:     []string{"customer-a"},
			FeatureIDsOrKeys: []string{featureByKey.Key},
		})
		require.NoError(t, err)
		require.Len(t, res.Items, 1)
		require.Equal(t, entA.ID, res.Items[0].ID)
	})

	t.Run("Should filter by customer key and feature ID", func(t *testing.T) {
		res, err := repo.entRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
			Namespaces:       []string{ns},
			CustomerKeys:     []string{"customer-b"},
			FeatureIDsOrKeys: []string{featureByID.ID},
		})
		require.NoError(t, err)
		require.Len(t, res.Items, 1)
		require.Equal(t, entB.ID, res.Items[0].ID)
	})

	t.Run("Should return empty result without querying entitlements when customer key does not exist", func(t *testing.T) {
		res, err := repo.entRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
			Namespaces:   []string{ns},
			CustomerKeys: []string{"missing-customer"},
		})
		require.NoError(t, err)
		require.Empty(t, res.Items)
		require.Zero(t, res.TotalCount)
	})
}

func TestListEntitlementsByIngestedEventsQuery(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		subject       string
		meters        []string
		expectedQuery string
		expectedArgs  []any
	}{
		{
			name:          "Should return entitlements for a subject",
			namespace:     "namespace-1",
			subject:       "subject-1",
			meters:        []string{"meter-1", "meter-2"},
			expectedQuery: `WITH "customer_by_subject" AS (SELECT "c"."id" FROM "customers" AS "c" WHERE "c"."namespace" = $1 AND "c"."key" = $2 AND "c"."deleted_at" IS NULL UNION SELECT "cs"."customer_id" FROM "customer_subjects" AS "cs" JOIN "customers" AS "c" ON "c"."id" = "cs"."customer_id" WHERE "cs"."namespace" = $3 AND "cs"."subject_key" = $4 AND "c"."deleted_at" IS NULL AND "cs"."deleted_at" IS NULL), "feature_by_meter" AS (SELECT "f"."id" FROM "features" AS "f" JOIN "meters" AS "m" ON "m"."id" = "f"."meter_id" WHERE "f"."namespace" = $5 AND "f"."archived_at" IS NULL AND "f"."deleted_at" IS NULL AND "m"."deleted_at" IS NULL AND "m"."key" IN ($6, $7)) SELECT "e"."namespace", "e"."id", "e"."created_at", "e"."deleted_at", "e"."active_from", "e"."active_to" FROM "entitlements" AS "e" JOIN "customer_by_subject" AS "cbs" ON "e"."customer_id" = "cbs"."id" JOIN "feature_by_meter" AS "fbm" ON "e"."feature_id" = "fbm"."id" WHERE "e"."namespace" = $8 AND "e"."deleted_at" IS NULL`,
			expectedArgs: []any{
				"namespace-1",
				"subject-1",
				"namespace-1",
				"subject-1",
				"namespace-1",
				"meter-1",
				"meter-2",
				"namespace-1",
			},
		},
		{
			name:          "Should return entitlements for a subject and meter",
			namespace:     "namespace-1",
			subject:       "subject-1",
			meters:        []string{"meter-1"},
			expectedQuery: `WITH "customer_by_subject" AS (SELECT "c"."id" FROM "customers" AS "c" WHERE "c"."namespace" = $1 AND "c"."key" = $2 AND "c"."deleted_at" IS NULL UNION SELECT "cs"."customer_id" FROM "customer_subjects" AS "cs" JOIN "customers" AS "c" ON "c"."id" = "cs"."customer_id" WHERE "cs"."namespace" = $3 AND "cs"."subject_key" = $4 AND "c"."deleted_at" IS NULL AND "cs"."deleted_at" IS NULL), "feature_by_meter" AS (SELECT "f"."id" FROM "features" AS "f" JOIN "meters" AS "m" ON "m"."id" = "f"."meter_id" WHERE "f"."namespace" = $5 AND "f"."archived_at" IS NULL AND "f"."deleted_at" IS NULL AND "m"."deleted_at" IS NULL AND "m"."key" IN ($6)) SELECT "e"."namespace", "e"."id", "e"."created_at", "e"."deleted_at", "e"."active_from", "e"."active_to" FROM "entitlements" AS "e" JOIN "customer_by_subject" AS "cbs" ON "e"."customer_id" = "cbs"."id" JOIN "feature_by_meter" AS "fbm" ON "e"."feature_id" = "fbm"."id" WHERE "e"."namespace" = $7 AND "e"."deleted_at" IS NULL`,
			expectedArgs: []any{
				"namespace-1",
				"subject-1",
				"namespace-1",
				"subject-1",
				"namespace-1",
				"meter-1",
				"namespace-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, args := adapter.EntitlementsByIngestedEventsQuery(dialect.Postgres, tt.namespace, tt.subject, tt.meters...)
			assert.Equal(t, tt.expectedQuery, q)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestListEntitlementsByIngestedEvents(t *testing.T) {
	repo, cleanup := setup(t)
	defer cleanup()

	ns := "namespace-1"

	m1, err := repo.meterRepo.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:   ns,
		Name:        "API Requests",
		Key:         "api_requests_total",
		Aggregation: meter.MeterAggregationCount,
		EventType:   "api-request",
		GroupBy: map[string]string{
			"method": "$.method",
			"path":   "$.path",
		},
	})
	require.NoErrorf(t, err, "meter creation should not fail")
	assert.NotEmptyf(t, m1.ID, "meter ID should not be empty")

	m2, err := repo.meterRepo.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     ns,
		Name:          "Tokens",
		Key:           "tokens_total",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "prompt",
		ValueProperty: lo.ToPtr("$.tokens"),
		GroupBy: map[string]string{
			"model": "$.model",
			"type":  "$.type",
		},
	})
	require.NoErrorf(t, err, "meter creation should not fail")
	assert.NotEmptyf(t, m2.ID, "meter ID should not be empty")

	f1, err := repo.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:                m1.Name,
		Key:                 m1.Key,
		Namespace:           m1.Namespace,
		MeterID:             lo.ToPtr(m1.ID),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(m1.GroupBy),
	})
	require.NoErrorf(t, err, "feature creation should not fail")
	assert.NotEmptyf(t, f1.ID, "feature ID should not be empty")

	f2, err := repo.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:                m2.Name,
		Key:                 m2.Key,
		Namespace:           m2.Namespace,
		MeterID:             lo.ToPtr(m2.ID),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(m2.GroupBy),
	})
	require.NoErrorf(t, err, "feature creation should not fail")
	assert.NotEmptyf(t, f2.ID, "feature ID should not be empty")

	c1, err := repo.customerRepo.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr("customer-1"),
			Name: "Customer 1",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-1"},
			},
		},
	})
	require.NoErrorf(t, err, "customer creation should not fail")
	assert.NotEmptyf(t, c1.ID, "customer ID should not be empty")
	require.NotEmptyf(t, c1.Key, "customer key should not be empty")
	require.NotEmptyf(t, c1.UsageAttribution, "customer usage attribution should not be empty")

	e1, err := repo.entRepo.CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        f1.ID,
		FeatureKey:       f1.Key,
		UsageAttribution: c1.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		MeasureUsageFrom: lo.ToPtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})),
		IsSoftLimit: lo.ToPtr(true),
	})
	require.NoErrorf(t, err, "entitlement creation should not fail")
	assert.NotEmptyf(t, e1.ID, "entitlement ID should not be empty")

	e2, err := repo.entRepo.CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
		Namespace:        ns,
		FeatureID:        f2.ID,
		FeatureKey:       f2.Key,
		UsageAttribution: c1.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		MeasureUsageFrom: lo.ToPtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})),
		IsSoftLimit: lo.ToPtr(true),
	})
	require.NoErrorf(t, err, "entitlement creation should not fail")
	assert.NotEmptyf(t, e2.ID, "entitlement ID should not be empty")

	balanceWorkerRepo, ok := repo.entRepo.(balanceworker.BalanceWorkerRepository)
	require.Truef(t, ok, "entitlement repository should implement BalanceWorkerRepository")
	require.NotNilf(t, balanceWorkerRepo, "balanceWorkerRepo should not be nil")

	expectedEntitlementIDs := []string{e1.ID, e2.ID}

	toEntitlementIDs := func(t *testing.T, entitlements []balanceworker.ListAffectedEntitlementsResponse) []string {
		t.Helper()

		result := make([]string, 0, len(entitlements))

		for _, e := range entitlements {
			result = append(result, e.EntitlementID)
		}

		return result
	}

	t.Run("Should return entitlements for customer key and meter", func(t *testing.T) {
		entitlements, err := balanceWorkerRepo.ListEntitlementsAffectedByIngestEvents(t.Context(), balanceworker.IngestEventQueryFilter{
			Namespace:    ns,
			EventSubject: lo.FromPtr(c1.Key),
			MeterSlugs:   []string{m1.Key, m2.Key},
		})
		require.NoErrorf(t, err, "entitlements should be fetched successfully")
		assert.NotEmptyf(t, entitlements, "entitlements should not be empty")

		assert.ElementsMatchf(t, expectedEntitlementIDs, toEntitlementIDs(t, entitlements), "entitlement IDs should match expected entitlement IDs")
	})

	t.Run("Should return entitlements for subject key and meters", func(t *testing.T) {
		entitlements, err := balanceWorkerRepo.ListEntitlementsAffectedByIngestEvents(t.Context(), balanceworker.IngestEventQueryFilter{
			Namespace:    ns,
			EventSubject: lo.FromPtr(c1.UsageAttribution).SubjectKeys[0],
			MeterSlugs:   []string{m1.Key, m2.Key},
		})
		require.NoErrorf(t, err, "entitlements should be fetched successfully")
		assert.NotEmptyf(t, entitlements, "entitlements should not be empty")

		assert.ElementsMatchf(t, expectedEntitlementIDs, toEntitlementIDs(t, entitlements), "entitlement IDs should match expected entitlement IDs")
	})

	t.Run("Should return no entitlements for non-existing features", func(t *testing.T) {
		entitlements, err := balanceWorkerRepo.ListEntitlementsAffectedByIngestEvents(t.Context(), balanceworker.IngestEventQueryFilter{
			Namespace:    ns,
			EventSubject: lo.FromPtr(c1.UsageAttribution).SubjectKeys[0],
			MeterSlugs:   []string{"non-existent-meter"},
		})
		require.NoErrorf(t, err, "entitlements should be fetched successfully")
		assert.Emptyf(t, entitlements, "entitlements should be empty")
	})

	t.Run("Should return only non-deleted entitlements", func(t *testing.T) {
		err = repo.entRepo.DeleteEntitlement(t.Context(), models.NamespacedID{
			Namespace: e2.Namespace,
			ID:        e2.ID,
		}, clock.Now())
		require.NoErrorf(t, err, "deleting entitlement should not fail")

		entitlements, err := balanceWorkerRepo.ListEntitlementsAffectedByIngestEvents(t.Context(), balanceworker.IngestEventQueryFilter{
			Namespace:    ns,
			EventSubject: lo.FromPtr(c1.UsageAttribution).SubjectKeys[0],
			MeterSlugs:   []string{m1.Key, m2.Key},
		})
		require.NoErrorf(t, err, "entitlements should be fetched successfully")
		assert.NotEmptyf(t, entitlements, "entitlements should not be empty")

		assert.ElementsMatchf(t, []string{e1.ID}, toEntitlementIDs(t, entitlements), "entitlement IDs should match expected entitlement IDs")
	})

	t.Run("Should return no entitlements for deleted customer", func(t *testing.T) {
		err = repo.customerRepo.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: c1.Namespace,
			ID:        c1.ID,
		})
		require.NoErrorf(t, err, "deleting entitlement should not fail")

		entitlements, err := balanceWorkerRepo.ListEntitlementsAffectedByIngestEvents(t.Context(), balanceworker.IngestEventQueryFilter{
			Namespace:    ns,
			EventSubject: lo.FromPtr(c1.UsageAttribution).SubjectKeys[0],
			MeterSlugs:   []string{m1.Key, m2.Key},
		})
		require.NoErrorf(t, err, "entitlements should be fetched successfully")
		assert.Emptyf(t, entitlements, "entitlements should be empty")
	})
}
