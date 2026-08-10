package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPrioritizeGrants(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expiresAfterHour := &grant.ExpirationPeriod{
		Duration: grant.ExpirationPeriodDurationHour,
		Count:    1,
	}

	grants := []grant.Grant{
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-4 * time.Hour)},
			ID:           "no-expiration",
			Priority:     1,
		},
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-4 * time.Hour)},
			ID:           "lower-priority",
			Priority:     2,
			EffectiveAt:  base.Add(-time.Hour),
			Expiration:   expiresAfterHour,
		},
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-2 * time.Hour)},
			ID:           "created-first",
			Priority:     1,
			EffectiveAt:  base.Add(900 * time.Millisecond),
			Expiration:   expiresAfterHour,
		},
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-time.Hour)},
			ID:           "cursor-b",
			Priority:     1,
			EffectiveAt:  base.Add(500 * time.Millisecond),
			Expiration:   expiresAfterHour,
		},
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-time.Hour)},
			ID:           "cursor-a",
			Priority:     1,
			EffectiveAt:  base.Add(100 * time.Millisecond),
			Expiration:   expiresAfterHour,
		},
		{
			ManagedModel: models.ManagedModel{CreatedAt: base.Add(-3 * time.Hour)},
			ID:           "later-expiration",
			Priority:     1,
			EffectiveAt:  base.Add(time.Hour),
			Expiration:   expiresAfterHour,
		},
	}

	engine.PrioritizeGrants(grants)

	grantIDs := make([]string, 0, len(grants))
	for _, grant := range grants {
		grantIDs = append(grantIDs, grant.ID)
	}

	require.Equal(t, []string{
		"created-first",
		"cursor-a",
		"cursor-b",
		"later-expiration",
		"no-expiration",
		"lower-priority",
	}, grantIDs)
}
