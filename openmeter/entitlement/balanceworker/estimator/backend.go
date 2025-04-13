package estimator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

type CacheBackend interface {
	UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error)) (*EntitlementCached, error)
	Delete(ctx context.Context, target TargetEntitlement) error
	Get(ctx context.Context, target TargetEntitlement) (*EntitlementCached, error)
}

type redisCacheBackend struct {
	redis *redis.Client

	redsync     *redsync.Redsync
	lockTimeout time.Duration
}

type RedisCacheBackendOptions struct {
	RedisURL    string
	LockTimeout time.Duration
}

func (o *RedisCacheBackendOptions) Validate() error {
	if o.RedisURL == "" {
		return errors.New("redisURL is required")
	}

	if o.LockTimeout <= 0 {
		return errors.New("lockTimeout must be greater than 0")
	}

	return nil
}

func NewRedisCacheBackend(in RedisCacheBackendOptions) (CacheBackend, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	redisURLParsed, err := redis.ParseURL(in.RedisURL)
	if err != nil {
		return nil, err
	}

	redis := redis.NewClient(redisURLParsed)

	return &redisCacheBackend{
		redis:       redis,
		redsync:     redsync.New(goredis.NewPool(redis)),
		lockTimeout: in.LockTimeout,
	}, nil
}

type cacheEntryStatus string

const (
	cacheEntryStatusPending cacheEntryStatus = "pending"
	cacheEntryStatusValid   cacheEntryStatus = "valid"
)

type wrappedCacheEntry struct {
	Status cacheEntryStatus `json:"status"`
	// To ensure that values are not equal, we use a random ULID
	RandomULID string             `json:"random"`
	Entry      *EntitlementCached `json:"entry"`
}

func (b *redisCacheBackend) withRedisLock(ctx context.Context, target TargetEntitlement, fn func(ctx context.Context, lock *redsync.Mutex) error) error {
	lockKey := fmt.Sprintf("lock:entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())

	lock := b.redsync.NewMutex(lockKey, redsync.WithExpiry(b.lockTimeout))

	if err := lock.LockContext(ctx); err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			lock.Unlock()
			panic(recovered)
		}
	}()

	fnErr := fn(ctx, lock)

	// Might return ErrLockAlreadyExpired
	if _, err := lock.UnlockContext(ctx); err != nil {
		// Function might have returned an error, in this case we want to return that error instead of joining
		// the errors (so that the caller can handle the specific callback error and not the lock error)
		if fnErr != nil {
			return fnErr
		}

		return err

	}

	return fnErr
}

func (b *redisCacheBackend) getCacheEntryKey(target TargetEntitlement) string {
	return fmt.Sprintf("entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())
}

func (b *redisCacheBackend) UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error)) (*EntitlementCached, error) {
	var newEntry *EntitlementCached

	err := b.withRedisLock(ctx, target, func(ctx context.Context, lock *redsync.Mutex) error {
		key := b.getCacheEntryKey(target)

		// TODO: add cache entry ttl as now the entry is always updated thus ttls resets
		// Let's delete the entry in case the lock expires, that way subsequent calls will assume cold cache
		entryJSON, err := b.redis.GetDel(ctx, key).Result()

		notCached := false
		if err != nil {
			if errors.Is(err, redis.Nil) {
				notCached = true
			} else {
				return err
			}
		}

		var existingEntry *EntitlementCached
		if !notCached {
			unmarshaled := EntitlementCached{}
			err = json.Unmarshal([]byte(entryJSON), &unmarshaled)
			if err != nil {
				return err
			}

			existingEntry = &unmarshaled
		}

		newEntry, err = updater(ctx, existingEntry)
		if err != nil {
			return err
		}

		// If the updater returns nil, we do not update the cache
		if newEntry == nil {
			return nil
		}

		newValue, err := json.Marshal(newEntry)
		if err != nil {
			return err
		}

		// We should only update the cache if the lock is not yet expired.
		// It is fine to do the update() call even if the lock is expired, as this might be a recalculation
		// that we would want to execute regardless. This is why we are creating the timeout context here.
		//
		// Let's create a child context with a timeout for the update.

		lockCtx, lockCancel := context.WithDeadline(ctx, lock.Until())
		defer lockCancel()

		_, err = b.redis.Set(lockCtx, key, newValue, time.Hour).Result()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return redsync.ErrLockAlreadyExpired
			}

			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntry, nil
}

func (b *redisCacheBackend) Delete(ctx context.Context, target TargetEntitlement) error {
	return b.withRedisLock(ctx, target, func(ctx context.Context, lock *redsync.Mutex) error {
		key := b.getCacheEntryKey(target)
		return b.redis.Del(ctx, key).Err()
	})
}

func (b *redisCacheBackend) Get(ctx context.Context, target TargetEntitlement) (*EntitlementCached, error) {
	unmarshaled := EntitlementCached{}

	err := b.withRedisLock(ctx, target, func(ctx context.Context, lock *redsync.Mutex) error {
		key := b.getCacheEntryKey(target)
		entryJSON, err := b.redis.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(entryJSON), &unmarshaled)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &unmarshaled, nil
}
