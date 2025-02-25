package customer

import (
	"context"
	"errors"
	"sync"

	"github.com/samber/lo"
)

type RequestValidatorKey string

type RequestValidator interface {
	ValidateDeleteCustomer(context.Context, DeleteCustomerInput) error
	ValidateCreateCustomer(context.Context, CreateCustomerInput) error
	ValidateUpdateCustomer(context.Context, UpdateCustomerInput) error
}

var _ RequestValidator = (*NoopRequestValidator)(nil)

type NoopRequestValidator struct{}

func (NoopRequestValidator) ValidateDeleteCustomer(context.Context, DeleteCustomerInput) error {
	return nil
}

func (NoopRequestValidator) ValidateCreateCustomer(context.Context, CreateCustomerInput) error {
	return nil
}

func (NoopRequestValidator) ValidateUpdateCustomer(context.Context, UpdateCustomerInput) error {
	return nil
}

type RequestValidatorRegistry interface {
	RequestValidator

	Register(RequestValidator)
}

func NewRequestValidatorRegistry() RequestValidatorRegistry {
	return &requestValidatorRegistry{}
}

type requestValidatorRegistry struct {
	mu         sync.RWMutex
	validators []RequestValidator
}

var (
	_ RequestValidator         = (*requestValidatorRegistry)(nil)
	_ RequestValidatorRegistry = (*requestValidatorRegistry)(nil)

	ErrValidatorAlreadyRegistered = errors.New("validator already registered")
)

func (r *requestValidatorRegistry) Register(v RequestValidator) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.validators = append(r.validators, v)
}

func (r *requestValidatorRegistry) ValidateDeleteCustomer(ctx context.Context, input DeleteCustomerInput) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return errors.Join(
		lo.Map(r.validators, func(v RequestValidator, _ int) error {
			return v.ValidateDeleteCustomer(ctx, input)
		})...,
	)
}

func (r *requestValidatorRegistry) ValidateCreateCustomer(ctx context.Context, input CreateCustomerInput) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return errors.Join(
		lo.Map(r.validators, func(v RequestValidator, _ int) error {
			return v.ValidateCreateCustomer(ctx, input)
		})...,
	)
}

func (r *requestValidatorRegistry) ValidateUpdateCustomer(ctx context.Context, input UpdateCustomerInput) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return errors.Join(
		lo.Map(r.validators, func(v RequestValidator, _ int) error {
			return v.ValidateUpdateCustomer(ctx, input)
		})...,
	)
}
