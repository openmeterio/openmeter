package customersentitlements

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestToAPIEntitlementGrant(t *testing.T) {
	now := time.Date(2026, 9, 16, 10, 30, 0, 0, time.UTC)
	effectiveAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := effectiveAt.AddDate(0, 3, 0)

	base := grant.Grant{
		ManagedModel: models.ManagedModel{
			CreatedAt: effectiveAt.Add(-time.Hour),
			UpdatedAt: effectiveAt.Add(-time.Hour),
		},
		NamespacedModel:  models.NamespacedModel{Namespace: "ns"},
		ID:               "01K4WAQ0J99ZZ0MD75HXR112HA",
		OwnerID:          "01K4WAQ0J99ZZ0MD75HXR112H9",
		Amount:           100.5,
		Priority:         2,
		EffectiveAt:      effectiveAt,
		ResetMaxRollover: 100.5,
		ResetMinRollover: 10,
	}

	t.Run("maps a never expiring non-recurring grant", func(t *testing.T) {
		got, err := ToAPIEntitlementGrant(base, now)
		require.NoError(t, err)

		require.Equal(t, base.ID, got.Id)
		require.Equal(t, base.OwnerID, got.EntitlementId)
		require.Equal(t, "100.5", got.Amount)
		require.Equal(t, uint8(2), got.Priority)
		require.Equal(t, effectiveAt, got.EffectiveAt)
		require.Equal(t, "100.5", got.MaxRolloverAmount)
		require.Equal(t, "10", got.MinRolloverAmount)
		require.Nil(t, got.ExpiresAfter)
		require.Nil(t, got.ExpiresAt)
		require.Nil(t, got.Recurrence)
		require.Nil(t, got.NextRecurrence)
		require.Nil(t, got.VoidedAt)
		require.Nil(t, got.DeletedAt)
		require.NotNil(t, got.Labels)
		require.Empty(t, *got.Labels)
	})

	t.Run("maps expiration, recurrence, labels and lifecycle timestamps", func(t *testing.T) {
		voidedAt := now.Add(-time.Hour)
		deletedAt := now.Add(-time.Minute)

		g := base
		g.Expiration = &grant.ExpirationPeriod{Count: 3, Duration: grant.ExpirationPeriodDurationMonth}
		g.ExpiresAt = &expiresAt
		g.Recurrence = &timeutil.Recurrence{Interval: timeutil.RecurrencePeriodWeek, Anchor: effectiveAt}
		g.Metadata = map[string]string{"source": "promo"}
		g.Annotations = models.Annotations{"subscription.id": "01K4WAQ0J99ZZ0MD75HXR112HB"}
		g.VoidedAt = &voidedAt
		g.DeletedAt = &deletedAt

		got, err := ToAPIEntitlementGrant(g, now)
		require.NoError(t, err)

		require.Equal(t, "P3M", lo.FromPtr(got.ExpiresAfter))
		require.Equal(t, expiresAt, lo.FromPtr(got.ExpiresAt))
		require.NotNil(t, got.Recurrence)
		require.Equal(t, "P1W", got.Recurrence.Interval)
		require.Equal(t, effectiveAt, got.Recurrence.Anchor)
		// The anchor is a Tuesday, so the first weekly occurrence not before now is
		// the following Tuesday.
		require.Equal(t, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC), lo.FromPtr(got.NextRecurrence))
		require.Equal(t, voidedAt, lo.FromPtr(got.VoidedAt))
		require.Equal(t, deletedAt, lo.FromPtr(got.DeletedAt))
		require.Equal(t, map[string]string{
			"source":                    "promo",
			"openmeter_subscription.id": "01K4WAQ0J99ZZ0MD75HXR112HB",
		}, map[string]string(*got.Labels))
	})
}

func TestToAPIEntitlementGrantExpiresAfter(t *testing.T) {
	cases := []struct {
		period grant.ExpirationPeriod
		want   string
	}{
		{grant.ExpirationPeriod{Count: 12, Duration: grant.ExpirationPeriodDurationHour}, "PT12H"},
		{grant.ExpirationPeriod{Count: 7, Duration: grant.ExpirationPeriodDurationDay}, "P7D"},
		{grant.ExpirationPeriod{Count: 2, Duration: grant.ExpirationPeriodDurationWeek}, "P2W"},
		{grant.ExpirationPeriod{Count: 3, Duration: grant.ExpirationPeriodDurationMonth}, "P3M"},
		{grant.ExpirationPeriod{Count: 1, Duration: grant.ExpirationPeriodDurationYear}, "P1Y"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got, err := toAPIEntitlementGrantExpiresAfter(tc.period)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("rejects an unknown unit", func(t *testing.T) {
		_, err := toAPIEntitlementGrantExpiresAfter(grant.ExpirationPeriod{Count: 1, Duration: "DECADE"})
		require.ErrorContains(t, err, "unsupported expiration duration")
	})
}

func TestFromAPIEntitlementGrantSortField(t *testing.T) {
	for _, field := range []string{"created_at", "updated_at", "effective_at", "expires_at"} {
		got, err := fromAPIEntitlementGrantSortField(t.Context(), field)
		require.NoError(t, err)
		require.Equal(t, grant.OrderBy(field), got)
	}

	// The owner is fixed by the path and the ID is not meaningful to sort by, so
	// neither is exposed even though the domain supports them.
	for _, field := range []string{"owner_id", "id", "amount"} {
		_, err := fromAPIEntitlementGrantSortField(t.Context(), field)
		require.ErrorContains(t, err, "unsupported sort field")
	}
}
