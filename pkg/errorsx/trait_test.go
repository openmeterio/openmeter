package errorsx_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type mErr struct {
	err error
}

func (m mErr) Error() string {
	return m.err.Error()
}

func (m mErr) Unwrap() error {
	return m.err
}

func TestTraits(t *testing.T) {
	t1 := errorsx.NewTrait("t1")
	e1 := errors.New("e1")

	assert.False(t, errorsx.HasTrait(e1, t1), "Should not have false positives")

	e2 := errorsx.WithTrait(e1, t1)
	assert.True(t, errorsx.HasTrait(e2, t1), "Should find trait of error")

	t2 := errorsx.NewTrait("t2")
	assert.False(t, errorsx.HasTrait(e2, t2), "Should not find trait not present")

	e3 := mErr{err: e2}
	assert.True(t, errorsx.HasTrait(e3, t1), "Should find trait of wrapped error")

	e4 := errorsx.WithTrait(e3, t2)
	assert.True(t, errorsx.HasTrait(e4, t1), "Should find trait of wrapped error")
	assert.True(t, errorsx.HasTrait(e4, t2), "Should find trait of wrapped error")

	e5 := errors.Join(e1, e2, e3, e4)
	assert.False(t, errorsx.HasTrait(e5, t1), "Does NOT parse joined errors")
}
