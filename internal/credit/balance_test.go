package credit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
)

func TestGrantBalanceMap(t *testing.T) {
	makeGrant := func(id string) credit.Grant {
		return credit.Grant{
			ID: id,
		}
	}

	t.Run("ExactlyForGrants", func(t *testing.T) {
		makeGrant("1")

		gbm := credit.GrantBalanceMap{
			"1": 100.0,
			"2": 100.0,
			"3": 100.0,
			"4": 100.0,
		}

		assert.True(t, gbm.ExactlyForGrants([]credit.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]credit.Grant{
			makeGrant("0"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]credit.Grant{
			makeGrant("1"),
			makeGrant("1"),
			makeGrant("3"),
			makeGrant("4"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]credit.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
			makeGrant("4"),
			makeGrant("5"),
		}))
		assert.False(t, gbm.ExactlyForGrants([]credit.Grant{
			makeGrant("1"),
			makeGrant("2"),
			makeGrant("3"),
		}))
	})
}
