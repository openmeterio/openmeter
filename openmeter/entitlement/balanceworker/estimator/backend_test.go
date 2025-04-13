package estimator

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TODO: parallel deletes must not fail and delete the cache key
// TODO: n parallel updates must fail
// TODO: parallel recalculations must not fail, but they should not update the cache

type NegCacheSuite struct {
	suite.Suite
	*require.Assertions

	redisURL string
}

func TestNegCache(t *testing.T) {
	suite.Run(t, new(NegCacheSuite))
}

func (s *NegCacheSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	if os.Getenv("OPENMETER_REDIS_URL") == "" {
		s.T().Skip("OPENMETER_REDIS_URL is not set, skipping test")
	}

	s.redisURL = os.Getenv("OPENMETER_REDIS_URL")
}

func (s *NegCacheSuite) TestBackendUpdateSanity() {
	// TODO: Set lock expiry to hours to avoid debug failures
	backend, err := NewRedisCacheBackend(RedisCacheBackendOptions{
		RedisURL:    s.redisURL,
		LockTimeout: 2 * time.Second,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	target := s.getTarget("update-empty-" + ulid.Make().String())

	// Given
	// - updating trying an entity that does not exists
	// When
	// - the calculation yeilds nil value
	// Then
	// - invokes the callback with empty entity
	// - the callback returns nil too
	// - the next update will still see empty state
	ent, err := backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		s.Require().Nil(entry)
		return nil, nil
	})
	s.Require().NoError(err)
	s.Require().Nil(ent)

	ent, err = backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		s.Require().Nil(entry)
		return nil, nil
	})
	s.Require().NoError(err)
	s.Require().Nil(ent)

	target = s.getTarget("update-valid-" + ulid.Make().String())

	// Given
	// - updating an entity that does not exists
	// When
	// - the calculation yields a snapshot
	// Then
	// - the cache is updated with the new snapshot
	ent, err = backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		s.Require().Nil(entry)
		return &EntitlementCached{
			LastCalculation: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
			},
			LastCalculated: time.Now(),
			ApproxUsage:    NewInfDecimal(11),
		}, nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(ent)
	s.Require().Equal(10.0, *ent.LastCalculation.Balance)

	// Given
	// - updating an entity that exists
	// When
	// - the calculation yields a snapshot
	// Then
	// - the cache is updated with the new snapshot
	ent, err = backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		s.Require().NotNil(entry)
		s.Require().Equal(10.0, *entry.LastCalculation.Balance)
		s.Require().Equal(11.0, entry.ApproxUsage.InexactFloat64())
		return &EntitlementCached{
			LastCalculation: snapshot.EntitlementValue{
				Balance: lo.ToPtr(20.0),
			},
			LastCalculated: time.Now(),
			ApproxUsage:    NewInfDecimal(21),
		}, nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(ent)
	s.Require().Equal(20.0, *ent.LastCalculation.Balance)
	s.Require().Equal(21.0, ent.ApproxUsage.InexactFloat64())

	// Given
	// - an entity exists
	// When
	// - deleting the entity
	// Then
	// - the next cache update will get nil cache item
	err = backend.Delete(ctx, target)
	s.Require().NoError(err)

	ent, err = backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		s.Require().Nil(entry)
		return nil, nil
	})
	s.Require().NoError(err)
	s.Require().Nil(ent)

	// Deleting a non-existing entity must not fail (deletes pending wrapper)
	err = backend.Delete(ctx, target)
	s.Require().NoError(err)

	// Deleting a non-existing entity must not fail (no key exists)
	err = backend.Delete(ctx, target)
	s.Require().NoError(err)
}

func (s *NegCacheSuite) TestBackendUpdateConcurrency() {
	backend, err := NewRedisCacheBackend(RedisCacheBackendOptions{
		RedisURL:    s.redisURL,
		LockTimeout: 2 * time.Second,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	target := s.getTarget("update-concurrency-" + ulid.Make().String())

	// Given
	// - an entity exists
	// When
	// - multiple updates are performed concurrently
	// Then
	// - all updates are properly sequenced

	_, err = backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		return &EntitlementCached{
			LastCalculation: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
			},
			LastCalculated: time.Now(),
			ApproxUsage:    NewInfDecimal(0),
		}, nil
	})
	s.Require().NoError(err)
	wg := sync.WaitGroup{}

	waitChan := make(chan struct{})

	nrThreads := 200

	results := make([]*EntitlementCached, nrThreads)
	resultsErr := make([]error, nrThreads)

	for i := 0; i < nrThreads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			out, err := backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
				<-waitChan

				return &EntitlementCached{
					LastCalculation: snapshot.EntitlementValue{
						Balance: lo.ToPtr(*entry.LastCalculation.Balance + 1.0),
					},
					LastCalculated: time.Now(),
					ApproxUsage:    NewInfDecimal(10),
				}, nil
			})
			results[i] = out
			resultsErr[i] = err
		}(i)
	}

	close(waitChan)
	wg.Wait()

	s.NoError(errors.Join(resultsErr...))

	// Getting the cache entry must return a consistent value
	ent, err := backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		return entry, nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(ent)
	s.Require().Equal(float64(nrThreads), *ent.LastCalculation.Balance)
}

func (s *NegCacheSuite) getTarget(id string) TargetEntitlement {
	return TargetEntitlement{
		Entitlement: entitlement.Entitlement{
			GenericProperties: entitlement.GenericProperties{
				ID: id,
				CurrentUsagePeriod: &timeutil.Period{
					From: time.Now(),
					To:   time.Now().Add(time.Hour),
				},
			},
		},
		Feature: feature.Feature{
			ID: "1",
		},
		Meter: meter.Meter{
			ManagedResource: models.ManagedResource{
				ID: "1",
			},
		},
	}
}
