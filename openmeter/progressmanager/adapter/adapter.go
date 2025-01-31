package adapter

import (
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/openmeter/progressmanager"
)

type Config struct {
	Expiration time.Duration
	Redis      *redis.Client
	Logger     *slog.Logger
}

func (c Config) Validate() error {
	if c.Expiration <= 0 {
		return errors.New("expiration must be greater than 0")
	}

	if c.Redis == nil {
		return errors.New("redis client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (progressmanager.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		expiration: config.Expiration,
		redis:      config.Redis,
		logger:     config.Logger,
	}, nil
}

var _ progressmanager.Adapter = (*adapter)(nil)

type adapter struct {
	expiration time.Duration
	redis      *redis.Client
	logger     *slog.Logger
}

func NewNoop() progressmanager.Adapter {
	return &adapterNoop{}
}

var _ progressmanager.Adapter = (*adapterNoop)(nil)

type adapterNoop struct{}
