package secretservice

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/lrux"
)

const (
	DefaultCacheSize = 10_000
	DefaultCacheTTL  = time.Minute
)

var _ secret.Service = (*Service)(nil)

type Service struct {
	adapter secret.Adapter
	cache   *lrux.CacheWithItemTTL[secretentity.SecretID, secretentity.Secret]
}

type Config struct {
	Adapter   secret.Adapter
	CacheSize int
	CacheTTL  time.Duration
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.CacheSize < 0 {
		return errors.New("cache size cannot be negative")
	}

	if c.CacheTTL < 0 {
		return errors.New("cache ttl cannot be negative")
	}

	return nil
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
