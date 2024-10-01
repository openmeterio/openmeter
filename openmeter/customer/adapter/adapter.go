package adapter

import (
	"errors"
	"log/slog"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (customer.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:        entcontext.NewClient(config.Client),
		logger:    config.Logger,
		observers: &[]appobserver.Observer[customerentity.Customer]{},
	}, nil
}

var (
	_ customer.Adapter                               = (*adapter)(nil)
	_ appobserver.Publisher[customerentity.Customer] = (*adapter)(nil)
)

type adapter struct {
	db entcontext.DB
	// It is a reference so we can pass it down in WithTx
	observers *[]appobserver.Observer[customerentity.Customer]

	logger *slog.Logger
}

func (a *adapter) DB() entcontext.DB {
	return a.db
}
