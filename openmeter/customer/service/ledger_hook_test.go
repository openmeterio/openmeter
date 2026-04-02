package customerservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customertestutils "github.com/openmeterio/openmeter/openmeter/customer/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomerService_CreateCustomerProvisionsLedgerAccounts(t *testing.T) {
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

	ledgerDeps, err := ledgertestutils.InitDeps(env.Client, env.Logger)
	require.NoError(t, err)

	hook, err := ledgerresolvers.NewCustomerLedgerHook(ledgerresolvers.CustomerLedgerHookConfig{
		Service: ledgerDeps.ResolversService,
		Tracer:  env.Tracer,
	})
	require.NoError(t, err)
	env.CustomerService.RegisterHooks(hook)

	namespace := customertestutils.NewTestNamespace(t)

	created, err := env.CustomerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr("acme-ledger"),
			Name: "ACME Ledger",
		},
	})
	require.NoError(t, err)

	accounts, err := ledgerDeps.ResolversService.GetCustomerAccounts(t.Context(), customer.CustomerID{
		Namespace: namespace,
		ID:        created.ID,
	})
	require.NoError(t, err)
	assert.NotNil(t, accounts.FBOAccount)
	assert.NotNil(t, accounts.ReceivableAccount)
	assert.NotNil(t, accounts.AccruedAccount)
}

func TestCustomerService_CreateCustomerRollsBackWhenLedgerProvisioningFails(t *testing.T) {
	env := customertestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

	boom := errors.New("boom")

	hook, err := ledgerresolvers.NewCustomerLedgerHook(ledgerresolvers.CustomerLedgerHookConfig{
		Service: failingCustomerAccountProvisioner{err: boom},
		Tracer:  noop.NewTracerProvider().Tracer("test"),
	})
	require.NoError(t, err)
	env.CustomerService.RegisterHooks(hook)

	namespace := customertestutils.NewTestNamespace(t)

	created, err := env.CustomerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr("acme-fail"),
			Name: "ACME Fail",
		},
	})
	require.ErrorIs(t, err, boom)
	assert.Nil(t, created)

	_, err = env.CustomerService.GetCustomer(t.Context(), customer.GetCustomerInput{
		CustomerKey: &customer.CustomerKey{
			Namespace: namespace,
			Key:       "acme-fail",
		},
	})

	var notFoundErr *models.GenericNotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

type failingCustomerAccountProvisioner struct {
	err error
}

func (f failingCustomerAccountProvisioner) CreateCustomerAccounts(_ context.Context, _ customer.CustomerID) (ledger.CustomerAccounts, error) {
	return ledger.CustomerAccounts{}, f.err
}
