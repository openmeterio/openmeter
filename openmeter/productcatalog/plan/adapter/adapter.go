package adapter

import (
	"errors"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

var _ plan.Validator = (*Config)(nil)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, errors.New("postgres client is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func New(config Config) (plan.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ plan.Repository = (*adapter)(nil)

type adapter struct {
	db *entdb.Client

	logger *slog.Logger
}
