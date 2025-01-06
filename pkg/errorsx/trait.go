package errorsx

import (
	"errors"
	"sync/atomic"
)

// Traits
var internalId uint64

func nextId() uint64 {
	return atomic.AddUint64(&internalId, 1)
}

type Trait struct {
	id    uint64
	label string
}

func NewTrait(label string) Trait {
	return Trait{id: nextId(), label: label}
}

// Errors with Traits

type ErrorWithTraits interface {
	error
	Traits() []Trait
}

type errorWithTraits struct {
	err   error
	trait []Trait
}

var _ ErrorWithTraits = errorWithTraits{}

func (e errorWithTraits) Traits() []Trait {
	return e.trait
}

func (e errorWithTraits) Error() string {
	return e.err.Error()
}

func (e errorWithTraits) Unwrap() error {
	return e.err
}

// Managing Traits
func HasTrait(e error, t Trait) bool {
	if e == nil {
		return false
	}

	// First, we check the current error
	if et, ok := e.(ErrorWithTraits); ok {
		for _, trait := range et.Traits() {
			if trait == t {
				return true
			}
		}
	}

	// Then, we attempt to unwrap the inner error
	if uw := errors.Unwrap(e); uw != nil {
		return HasTrait(uw, t)
	} else if je, ok := e.(interface{ Unwrap() []error }); ok {
		for _, err := range je.Unwrap() {
			if HasTrait(err, t) {
				return true
			}
		}
	}

	return false
}

func WithTrait(err error, trait Trait) error {
	if err == nil {
		return nil
	}

	return errorWithTraits{err: err, trait: []Trait{trait}}
}
