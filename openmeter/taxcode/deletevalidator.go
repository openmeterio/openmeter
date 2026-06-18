package taxcode

import (
	"context"
	"errors"
	"sync"

	"github.com/samber/lo"
)

type DeleteValidator interface {
	ValidateDeleteTaxCode(ctx context.Context, input DeleteTaxCodeInput) error
}

var _ DeleteValidator = (*NoopDeleteValidator)(nil)

type NoopDeleteValidator struct{}

func (NoopDeleteValidator) ValidateDeleteTaxCode(context.Context, DeleteTaxCodeInput) error {
	return nil
}

type DeleteValidatorRegistry interface {
	DeleteValidator

	Register(DeleteValidator)
}

func NewDeleteValidatorRegistry() DeleteValidatorRegistry {
	return &deleteValidatorRegistry{}
}

type deleteValidatorRegistry struct {
	mu         sync.RWMutex
	validators []DeleteValidator
}

var (
	_ DeleteValidator         = (*deleteValidatorRegistry)(nil)
	_ DeleteValidatorRegistry = (*deleteValidatorRegistry)(nil)
)

func (r *deleteValidatorRegistry) Register(v DeleteValidator) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.validators = append(r.validators, v)
}

func (r *deleteValidatorRegistry) ValidateDeleteTaxCode(ctx context.Context, input DeleteTaxCodeInput) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return errors.Join(
		lo.Map(r.validators, func(v DeleteValidator, _ int) error {
			return v.ValidateDeleteTaxCode(ctx, input)
		})...,
	)
}
