package customer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

const (
	TestNamespace = "default"

	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	CustomerAdapter() customer.Adapter
	Customer() customer.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	customerAdapter customer.Adapter
	customer        customer.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) CustomerAdapter() customer.Adapter {
	return n.customerAdapter
}

func (n testEnv) Customer() customer.Service {
	return n.customer
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("customer")

	postgresHost := defaultx.IfZero(os.Getenv("POSTGRES_HOST"), DefaultPostgresHost)

	postgresDriver, err := pgdriver.NewPostgresDriver(ctx, fmt.Sprintf(PostgresURLTemplate, postgresHost))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres driver: %w", err)
	}

	entPostgresDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())
	entClient := entPostgresDriver.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = entClient.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create database schema: %w", err)
	}

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

		if err = entPostgresDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = postgresDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close postgres driver: %w", err))
		}

		return errs
	}

	return &testEnv{
		customerAdapter: customerAdapter,
		customer:        customerService,
		closerFunc:      closerFunc,
	}, nil
}
