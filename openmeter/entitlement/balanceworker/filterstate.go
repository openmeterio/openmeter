package balanceworker

import (
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/pkg/models"
)

type FilterStateStorageDriver string

const (
	FilterStateStorageDriverRedis    FilterStateStorageDriver = "redis"
	FilterStateStorageDriverInMemory FilterStateStorageDriver = "in-memory"
)

var _ models.Validator = (*FilterStateStorage)(nil)

type FilterStateStorage struct {
	driver FilterStateStorageDriver
	redis  *FilterStateStorageRedis
}

func NewFilterStateStorage[T FilterStateStorageRedis | FilterStateStorageInMemory](storage T) (FilterStateStorage, error) {
	switch driver := any(storage).(type) {
	case FilterStateStorageRedis:
		if err := driver.Validate(); err != nil {
			return FilterStateStorage{}, fmt.Errorf("redis: %w", err)
		}

		return FilterStateStorage{driver: FilterStateStorageDriverRedis, redis: &driver}, nil
	case FilterStateStorageInMemory:
		if err := driver.Validate(); err != nil {
			return FilterStateStorage{}, fmt.Errorf("in-memory: %w", err)
		}

		return FilterStateStorage{driver: FilterStateStorageDriverInMemory}, nil
	}

	return FilterStateStorage{}, fmt.Errorf("unsupported driver: %T", storage)
}

func (c FilterStateStorage) Validate() error {
	var errs []error

	if c.driver == "" {
		errs = append(errs, errors.New("driver is required"))
	}

	if c.driver == FilterStateStorageDriverRedis {
		if c.redis == nil {
			errs = append(errs, errors.New("redis is required"))
		} else {
			if err := c.redis.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("redis: %w", err))
			}
		}
	}

	return errors.Join(errs...)
}

func (c FilterStateStorage) Driver() FilterStateStorageDriver {
	return c.driver
}

func (c FilterStateStorage) Redis() (FilterStateStorageRedis, error) {
	if c.driver != FilterStateStorageDriverRedis {
		return FilterStateStorageRedis{}, fmt.Errorf("driver is not redis")
	}

	if c.redis == nil {
		return FilterStateStorageRedis{}, fmt.Errorf("redis is not initialized")
	}

	return *c.redis, nil
}

type FilterStateStorageRedis struct {
	Client     *redis.Client
	Expiration time.Duration
}

var _ models.Validator = (*FilterStateStorageRedis)(nil)

func (c FilterStateStorageRedis) Validate() error {
	if c.Client == nil {
		return errors.New("client is required")
	}

	if c.Expiration <= 0 {
		return errors.New("expiration must be greater than 0")
	}

	return nil
}

type FilterStateStorageInMemory struct{}

var _ models.Validator = (*FilterStateStorageInMemory)(nil)

func (c FilterStateStorageInMemory) Validate() error {
	return nil
}
