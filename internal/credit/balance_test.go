// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
