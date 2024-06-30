package postgresadapter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestActiveGrants(t *testing.T) {
	driver := testutils.InitPostgresDB(t)
	dbClient := db.NewClient(db.Driver(driver))
	defer dbClient.Close()

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	grantRepo := postgresadapter.NewPostgresGrantRepo(dbClient)
	ctx := context.Background()

	// Let's create a few grants for testing
	//		Overal setup:
	//		- g1: 2024-06-30 15:39:00+00 ... 2025-06-30 15:39:00+00
	//		- g2: 2024-06-28 14:38:00+00 ... 2024-06-29 14:38:00+00
	//		- g3: 2024-06-28 14:35:00+00 ... 2024-06-28 14:38:00+00

	_, err := grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
		OwnerID:     "owner-1",
		Namespace:   "namespace-1",
		Metadata:    map[string]string{"name": "g1"},
		Amount:      100,
		Priority:    1,
		EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"),
		Expiration: credit.ExpirationPeriod{
			Count:    1,
			Duration: credit.ExpirationPeriodDurationYear,
		},
		ExpiresAt: testutils.GetRFC3339Time(t, "2025-06-30T15:39:00Z"),
	})
	assert.NoError(t, err)

	grant2, err := grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
		OwnerID:     "owner-1",
		Namespace:   "namespace-1",
		Metadata:    map[string]string{"name": "g2"},
		Amount:      200,
		Priority:    3,
		EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:38:00Z"),
		Expiration: credit.ExpirationPeriod{
			Count:    1,
			Duration: credit.ExpirationPeriodDurationDay,
		},
		ExpiresAt:        testutils.GetRFC3339Time(t, "2024-06-29T14:38:00Z"),
		ResetMaxRollover: 20,
		Recurrence: &recurrence.Recurrence{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-06-28T14:38:00Z"),
		},
	})
	assert.NoError(t, err)

	grant3, err := grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
		OwnerID:     "owner-1",
		Namespace:   "namespace-1",
		Metadata:    map[string]string{"name": "g3"},
		Amount:      10,
		Priority:    5,
		EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"),
		Expiration: credit.ExpirationPeriod{
			Count:    1,
			Duration: credit.ExpirationPeriodDurationDay,
		},
		ExpiresAt: testutils.GetRFC3339Time(t, "2024-06-28T14:38:00Z"),
	})
	assert.NoError(t, err)

	// Test data is done

	// case: before active grants => no grants
	t.Run("before active grants", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2023-06-28T14:30:00Z"), testutils.GetRFC3339Time(t, "2024-06-28T14:33:59Z"))

		assert.NoError(t, err)
		assert.Empty(t, grants)
	})

	// case: after all is inactive => no grants
	t.Run("after all is inactive", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2026-06-28T14:30:00Z"), testutils.GetRFC3339Time(t, "2027-06-28T14:33:59Z"))

		assert.NoError(t, err)
		assert.Empty(t, grants)
	})

	// case: g3 is active 2024-06-28T14:35:00Z ... 2024-06-28T14:35:00Z
	t.Run("one active grant at the beginning of the period", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"), testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"))

		assert.NoError(t, err)
		assert.Len(t, grants, 1)
		assertGrantID(t, grant3, grants)
	})

	// case: g3 is active 2024-06-28T14:35:00Z ... 2024-06-28T14:37:00Z
	t.Run("one active grant at the beginning of the period, just before activation", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"), testutils.GetRFC3339Time(t, "2024-06-28T14:37:00Z"))

		assert.NoError(t, err)
		assert.Len(t, grants, 1)
		assertGrantID(t, grant3, grants)
	})

	// case: g3, g2 is active 2024-06-28T14:35:00Z ... 2024-06-28T14:38:00Z
	t.Run("two active grants at the beginning of the period", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2024-06-28T14:30:00Z"), testutils.GetRFC3339Time(t, "2024-06-28T14:38:00Z"))

		assert.NoError(t, err)
		assert.Len(t, grants, 2)
		assertGrantID(t, grant2, grants)
		assertGrantID(t, grant3, grants)
	})

	// case: g2 is active, requested period is between the effective at and expiration
	t.Run("one active grant, requested period is between the effective at and expiration", func(t *testing.T) {
		grants, err := grantRepo.ListActiveGrantsBetween(ctx, credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        "owner-1",
		}, testutils.GetRFC3339Time(t, "2024-06-28T15:30:00Z"), testutils.GetRFC3339Time(t, "2024-06-28T15:39:01Z"))

		assert.NoError(t, err)
		assert.Len(t, grants, 1)
		assertGrantID(t, grant2, grants)
	})

}

func assertGrantID(t *testing.T, grant *credit.Grant, res []credit.Grant) {
	for _, g := range res {
		if g.ID == grant.ID {
			return
		}
	}
	t.Errorf("grant %s not found", grant.ID)
}
