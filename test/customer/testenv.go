package customer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	Customer() customer.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	customer customer.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) Customer() customer.Service {
	return n.customer
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("customer")

	// Initialize postgres driver
	driver := testutils.InitPostgresDB(t)

	entClient := driver.EntDriver.Client()
	if err := migrate.Up(driver.URL); err != nil {
		t.Fatalf("failed to migrate db: %s", err.Error())
	}

	// Initialize customer adapter
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: entClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter: customerAdapter,
	})
	if err != nil {
		return nil, err
	}

	closerFunc := func() error {
		var errs error

		if err = entClient.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = driver.PGDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close postgres driver: %w", err))
		}

		return errs
	}

	return &testEnv{
		customer:   customerService,
		closerFunc: closerFunc,
	}, nil
}
