package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Client         *entdb.Client
	BillingAdapter billing.StandardInvoiceAdapter
}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, errors.New("ent client is required"))
	}

	if c.BillingAdapter == nil {
		errs = append(errs, errors.New("billing adapter is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (invoicemetrics.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:             config.Client,
		billingAdapter: config.BillingAdapter,
	}, nil
}

var _ invoicemetrics.Adapter = (*adapter)(nil)

type adapter struct {
	db             *entdb.Client
	billingAdapter billing.StandardInvoiceAdapter
}

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txDB := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &adapter{
		db:             txDB.Client(),
		billingAdapter: a.billingAdapter,
	}
}

func (a *adapter) Self() *adapter {
	return a
}
