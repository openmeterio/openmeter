package secretservice

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/lrux"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	DefaultCacheSize = 10_000
	DefaultCacheTTL  = time.Minute

	cacheLockShards = 256
)

var _ secret.Service = (*Service)(nil)

type Service struct {
	adapter secret.Adapter
	cache   *lrux.CacheWithItemTTL[secretentity.SecretID, secretentity.Secret]
	locks   [cacheLockShards]sync.Mutex
}

func (s *Service) lockFor(id secretentity.SecretID) *sync.Mutex {
	hash := fnv.New32a()

	for _, part := range []string{id.Namespace, id.ID, id.AppID.ID, id.Key} {
		_, _ = hash.Write([]byte(part))
	}

	return &s.locks[hash.Sum32()%cacheLockShards]
}

type Config struct {
	Adapter   secret.Adapter
	CacheSize int
	CacheTTL  time.Duration
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.CacheSize < 0 {
		errs = append(errs, fmt.Errorf("cache size must not be negative: %d", c.CacheSize))
	}

	if c.CacheTTL < 0 {
		errs = append(errs, fmt.Errorf("cache ttl must not be negative: %s", c.CacheTTL))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cacheSize := config.CacheSize
	if cacheSize == 0 {
		cacheSize = DefaultCacheSize
	}

	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = DefaultCacheTTL
	}

	cache, err := lrux.NewCacheWithItemTTL(
		cacheSize,
		func(ctx context.Context, id secretentity.SecretID) (secretentity.Secret, error) {
			return config.Adapter.GetAppSecret(ctx, id)
		},
		lrux.WithTTL(cacheTTL),
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		cache:   cache,
	}, nil
}
