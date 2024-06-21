package testutils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	assert.EventuallyWithTf(t, func(c *assert.CollectT) {
		fn(c, saveErr)
	}, wait, interval, "%w", firstVal(sm.Load(errKey)))
}
