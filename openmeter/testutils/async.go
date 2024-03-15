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

package testutils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func EventuallyWithTf(t *testing.T, fn func(c *assert.CollectT, saveErr func(err any)), wait time.Duration, interval time.Duration) {
	errKey := "error"
	sm := sync.Map{}
	saveErr := func(err any) {
		sm.Store(errKey, err)
	}

	firstVal := func(v ...any) any {
		return v[0]
	}

	require.EventuallyWithTf(t, func(c *assert.CollectT) {
		fn(c, saveErr)
	}, wait, interval, "%w", firstVal(sm.Load(errKey)))
}
