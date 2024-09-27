package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

const (
	TestNamespace = "default"

	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	Adapter() app.Adapter
	App() app.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	adapter app.Adapter
	app     app.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) Adapter() app.Adapter {
	return n.adapter
}

func (n testEnv) App() app.Service {
	return n.app
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(ctx context.Context) (TestEnv, error) {
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

	adapter, err := appadapter.New(appadapter.Config{
		Client: entClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	// TODO: we need to register the integration into a registry!
	service, err := appservice.New(appservice.Config{
		Adapter: adapter,
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
		adapter:    adapter,
		app:        service,
		closerFunc: closerFunc,
	}, nil
}
