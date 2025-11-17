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
	KeyPrefix  string
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
		keyPrefix:  config.KeyPrefix,
	}, nil
}

var _ progressmanager.Adapter = (*adapter)(nil)

type adapter struct {
	// keyPrefix is the prefix for progress data in the Redis store, if needed, the key format will be "<keyPrefix>:progress:<namespace>:<id>" or "progress:<namespace>:<id>" if the prefix is empty
	keyPrefix string
	// expiration defines how long progress data is stored in Redis before automatic removal
	expiration time.Duration
	// redis is the client for storing and retrieving progress data
	redis *redis.Client
	// logger is used for logging errors and debug information
	logger *slog.Logger
}

// NewNoop creates a no-operation adapter that implements the progressmanager.Adapter interface
// but performs no actual operations. This is useful for testing or when progress tracking
// is disabled.
func NewNoop() progressmanager.Adapter {
	return &adapterNoop{}
}

var _ progressmanager.Adapter = (*adapterNoop)(nil)

type adapterNoop struct{}
