package redisdedupe

import (
	"os"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HashdedupeTestSuite struct {
	suite.Suite
	*require.Assertions
	dedupe *HashDeduplicator
}

func TestHashDeduplicator(t *testing.T) {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")

	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}

	suite.Run(t, new(HashdedupeTestSuite))
}

func (s *HashdedupeTestSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	redisAddr := os.Getenv("TEST_REDIS_ADDR")

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	s.dedupe = NewHashDeduplicator(redisClient, time.Hour)
}

func (s *HashdedupeTestSuite) TearDownTest() {
	s.dedupe.Close()
}

func (s *HashdedupeTestSuite) TestSanity() {
	existingItems := generateRandomDedupeItems("ns-test", 10)

	ctx := s.T().Context()

	dup, err := s.dedupe.Set(ctx, existingItems...)
	s.NoError(err)
	s.Equal(0, len(dup))

	newItems := generateRandomDedupeItems("ns-test-2", 10)

	allItems := make([]dedupe.Item, 0, len(existingItems)+len(newItems))
	allItems = append(allItems, existingItems...)
	allItems = append(allItems, newItems...)

	batchResult, err := s.dedupe.CheckUniqueBatch(ctx, allItems)
	s.NoError(err)
	s.ElementsMatch(newItems, lo.Keys(batchResult.UniqueItems))
	s.ElementsMatch(existingItems, lo.Keys(batchResult.AlreadyProcessedItems))
}

func generateRandomDedupeItems(ns string, n int) []dedupe.Item {
	items := make([]dedupe.Item, n)
	for i := 0; i < n; i++ {
		items[i] = dedupe.Item{
			Namespace: ns,
			ID:        ulid.Make().String(),
			Source:    "test",
		}
	}
	return items
}
