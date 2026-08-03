package balance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	balance "github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
)

func TestNewStartingSnapshot(t *testing.T) {
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	activeGrant := grant.Grant{
		ID:          "active",
		Amount:      100,
		EffectiveAt: at,
	}
	futureGrant := grant.Grant{
		ID:          "future",
		Amount:      200,
		EffectiveAt: at.Add(time.Hour),
	}

	snapshot := balance.NewStartingSnapshot([]grant.Grant{activeGrant, futureGrant}, at)

	assert.Equal(t, at, snapshot.At)
	assert.Equal(t, balance.Map{
		activeGrant.ID: activeGrant.Amount,
		futureGrant.ID: 0,
	}, snapshot.Balances)
	assert.Zero(t, snapshot.Overage)
	assert.Equal(t, balance.SnapshottedUsage{
		Since: at,
		Usage: 0,
	}, snapshot.Usage)
	require.NotNil(t, snapshot.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{}, *snapshot.UsageSnapshot)
}

func TestGrantBalanceMap(t *testing.T) {
	makeGrant := func(id string) grant.Grant {
		return grant.Grant{
			ID: id,
		}
	}

	t.Run("ExactlyForGrants", func(t *testing.T) {
		makeGrant("1")

		gbm := balance.Map{
			"1": 100.0,
			"2": 100.0,
			"3": 100.0,
			"4": 100.0,
		}

		assert.True(t, gbm.ExactlyForGrants([]grant.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]grant.Grant{
			makeGrant("0"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]grant.Grant{
			makeGrant("1"),
			makeGrant("1"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]grant.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
			makeGrant("5"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]grant.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
		}))
	})
}

func TestSnapshotCloneCopiesUsageSnapshot(t *testing.T) {
	snapshot := balance.Snapshot{
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           5,
			TotalGrantUsage: 10,
		},
	}

	cloned := snapshot.Clone()
	require.NotNil(t, cloned.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           5,
		TotalGrantUsage: 10,
	}, *cloned.UsageSnapshot)

	cloned.UsageSnapshot.Usage = 15
	cloned.UsageSnapshot.TotalGrantUsage = 20
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           5,
		TotalGrantUsage: 10,
	}, *snapshot.UsageSnapshot)
}
