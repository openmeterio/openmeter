package billingservice

import (
	"context"
	"sync"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/samber/lo"
)

var _ billing.InvoiceLifecycleHandler = (*DefaultInvoiceLifecycleHandler)(nil)

type DefaultInvoiceLifecycleHandler struct{}

func (h *DefaultInvoiceLifecycleHandler) AreLinesBillable(ctx context.Context, invoice billing.GatheringInvoice, lines billing.GatheringLines) (billing.AreLinesBillableResult, error) {
	return lo.Map(lines, func(line billing.GatheringLine, _ int) billing.IsLineBillableResult {
		return billing.IsLineBillableResult{
			IsBillable:      true,
			ValidationError: nil,
		}
	}), nil
}

type LineLifecycleRegistry struct {
	mux            sync.RWMutex
	handlersByName map[billing.LifecycleHandler]billing.InvoiceLifecycleHandler
}

func NewLineLifecycleRegistry() *LineLifecycleRegistry {
	return &LineLifecycleRegistry{
		handlersByName: map[billing.LifecycleHandler]billing.InvoiceLifecycleHandler{
			billing.DefaultLifecycleHandler: &DefaultInvoiceLifecycleHandler{},
		},
	}
}

func (r *LineLifecycleRegistry) Register(typeName billing.LifecycleHandler, handler billing.InvoiceLifecycleHandler) {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.handlersByName[typeName] = handler
}

func (r *LineLifecycleRegistry) Get(ctx context.Context, line billing.LineWithInvoiceHeader) (billing.InvoiceLineLifecycleHandler, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	for _, handler := range r.handlers {
		shouldHandle, err := handler.ShouldHandleLine(ctx, line)
		if err != nil {
			return nil, err
		}

		if shouldHandle {
			return handler, nil
		}
	}

	return r.defaultHandler, nil
}
