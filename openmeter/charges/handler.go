package charges

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type UsageBasedRealizationInput struct {
	Charge       Charge
	AsOf         time.Time
	CurrentUsage *billing.StandardLine
}

// Handler is responsible for synchronizing the charge with external systems.
type Handler interface {
	OnStandardInvoiceRealizationCreated(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error)
	OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error)
	OnStandardInvoiceRealizationSettled(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error)
	OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input UsageBasedRealizationInput) ([]CreditRealizationCreateInput, error)
}

// Hooks router is a router that routes hooks to the appropriate hook implementation based on the charge.
// This allows for different hook implementation based on charge type/settlement modes/etc.

type HandlerRouterGetter func(charge Charge) (Handler, error)

type HandlerRouter struct {
	getter HandlerRouterGetter
}

var _ Handler = (*HandlerRouter)(nil)

func NewNoopHandlerRouter() *HandlerRouter {
	return &HandlerRouter{
		getter: func(charge Charge) (Handler, error) {
			return NoOpHandler{}, nil
		},
	}
}

func NewHandlerRouter(getter HandlerRouterGetter) (*HandlerRouter, error) {
	if getter == nil {
		return nil, errors.New("getter is required")
	}

	return &HandlerRouter{getter: getter}, nil
}

func (r *HandlerRouter) OnStandardInvoiceRealizationCreated(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	h, err := r.getter(charge)
	if err != nil {
		return charge, err
	}

	return h.OnStandardInvoiceRealizationCreated(ctx, charge, realization)
}

func (r *HandlerRouter) OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	h, err := r.getter(charge)
	if err != nil {
		return charge, err
	}

	return h.OnStandardInvoiceRealizationAuthorized(ctx, charge, realization)
}

func (r *HandlerRouter) OnStandardInvoiceRealizationSettled(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	h, err := r.getter(charge)
	if err != nil {
		return charge, err
	}

	return h.OnStandardInvoiceRealizationSettled(ctx, charge, realization)
}

func (r *HandlerRouter) OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input UsageBasedRealizationInput) ([]CreditRealizationCreateInput, error) {
	h, err := r.getter(input.Charge)
	if err != nil {
		return nil, err
	}

	return h.OnRealizeUsageBasedCreditChargePeriodically(ctx, input)
}

var _ Handler = (*NoOpHandler)(nil)

type NoOpHandler struct{}

func (h NoOpHandler) OnStandardInvoiceRealizationCreated(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	return charge, nil
}

func (h NoOpHandler) OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	return charge, nil
}

func (h NoOpHandler) OnStandardInvoiceRealizationSettled(ctx context.Context, charge Charge, realization StandardInvoiceRealizationWithLine) (Charge, error) {
	return charge, nil
}

func (h NoOpHandler) OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input UsageBasedRealizationInput) ([]CreditRealizationCreateInput, error) {
	return nil, nil
}
