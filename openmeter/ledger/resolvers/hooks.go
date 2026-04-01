package resolvers

import (
	"context"

	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// TODO: these hooks should be added to customer lifecycle
	CustomerLedgerHook     = models.ServiceHook[customer.Customer]
	NoopCustomerLedgerHook = models.NoopServiceHook[customer.Customer]
)

type customerAccountProvisioner interface {
	CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error)
}

type CustomerLedgerHookConfig struct {
	Service customerAccountProvisioner
	Tracer  trace.Tracer
}

// When we create a customer, we need to create the corresponding ledger accounts. Otherwise changes in the customer do not effect the ledger.
type customerLedgerHook struct {
	NoopCustomerLedgerHook

	config CustomerLedgerHookConfig
}

func NewCustomerLedgerHook(config CustomerLedgerHookConfig) (CustomerLedgerHook, error) {
	return &customerLedgerHook{
		NoopCustomerLedgerHook: NoopCustomerLedgerHook{},
		config:                 config,
	}, nil
}

func (h *customerLedgerHook) PostCreate(ctx context.Context, cust *customer.Customer) error {
	ctx, span := h.config.Tracer.Start(ctx, "customer_ledger_hook.post_create")
	defer span.End()

	_, err := h.config.Service.CreateCustomerAccounts(ctx, customer.CustomerID{
		Namespace: cust.Namespace,
		ID:        cust.ID,
	})
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to create customer accounts")
		span.RecordError(err)

		return err
	}

	return nil
}
