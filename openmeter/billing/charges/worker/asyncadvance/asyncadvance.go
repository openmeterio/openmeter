package chargesasyncadvance

import (
	"context"
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type Handler struct {
	chargesService charges.ChargeService
	logger         *slog.Logger
}

type Config struct {
	Logger         *slog.Logger
	ChargesService charges.ChargeService
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.ChargesService == nil {
		return errors.New("charges service is required")
	}

	return nil
}

func New(c Config) (*Handler, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &Handler{
		chargesService: c.ChargesService,
		logger:         c.Logger,
	}, nil
}

func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error {
	_, err := h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customer.CustomerID{
			Namespace: event.Namespace,
			ID:        event.CustomerID,
		},
	})
	if err != nil {
		h.logger.WarnContext(ctx, "failed to advance charges",
			slog.String("namespace", event.Namespace),
			slog.String("customer_id", event.CustomerID),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}
