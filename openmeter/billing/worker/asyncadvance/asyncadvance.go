package asyncadvance

import (
	"context"
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Handler struct {
	billingService billing.Service
	logger         *slog.Logger
}

type Config struct {
	Logger         *slog.Logger
	BillingService billing.Service
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.BillingService == nil {
		return errors.New("billing service is required")
	}

	if c.BillingService.GetAdvancementStrategy() != billing.ForegroundAdvancementStrategy {
		return errors.New("billing service must have foreground advancement strategy or we are creating an infinite loop")
	}

	return nil
}

func New(c Config) (*Handler, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &Handler{
		billingService: c.BillingService,
		logger:         c.Logger,
	}, nil
}

func (h *Handler) Handle(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error {
	_, err := h.billingService.AdvanceInvoice(ctx, event.Invoice)

	if errors.Is(err, billing.ErrInvoiceCannotAdvance) {
		h.logger.WarnContext(ctx, "invoice cannot advance (most probably a late message has occurred)", "invoice_id", event.Invoice)
		return nil
	}

	return err
}
