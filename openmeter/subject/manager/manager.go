package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubject "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
)

// Config is the configuration for the subject manager
type Config struct {
	CacheReloadInterval time.Duration
	CacheReloadTimeout  time.Duration
	CachePrefillCount   int
	CacheSize           int
	Ent                 *db.Client
	Logger              *slog.Logger
	PaginationSize      int
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.CacheReloadInterval <= 0 {
		return errors.New("cache reload interval must be greater than 0")
	}

	if c.CacheReloadTimeout <= 0 {
		return errors.New("cache reload timeout must be greater than 0")
	}

	if c.CachePrefillCount <= 0 {
		return errors.New("cache prefill count must be greater than 0")
	}

	if c.CachePrefillCount > c.CacheSize {
		return errors.New("cache prefill count must be less than or equal to cache size")
	}

	if c.CacheSize <= 0 {
		return errors.New("cache size must be greater than 0")
	}

	if c.Ent == nil {
		return errors.New("ent client is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.PaginationSize <= 0 {
		return errors.New("subject pagination size must be greater than 0")
	}

	return nil
}

// SubjectRef references a subject
type SubjectRef struct {
	Namespace string
	Key       string
}

// NewManager creates a new subject manager
func NewManager(config *Config) (*Manager, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	manager := &Manager{
		ent:                 config.Ent,
		cache:               expirable.NewLRU[string, struct{}](config.CacheSize, nil, 0),
		cacheReloadInterval: config.CacheReloadInterval,
		cacheReloadTimeout:  config.CacheReloadTimeout,
		cachePrefillCount:   config.CachePrefillCount,
		cacheSize:           config.CacheSize,
		logger:              config.Logger.WithGroup("subject-manager"),
		paginationSize:      config.PaginationSize,
	}

	// Initialize cache and schedule next reload
	config.Logger.Info("preheating subject manager cache", "cache_size", config.CacheSize, "cache_prefill_count", config.CachePrefillCount)
	start := time.Now()
	manager.reloadCache(reloadPrefill)
	config.Logger.Info("subject manager cache preheated", "duration.seconds", time.Since(start).Seconds())

	return manager, nil
}

// Manager is a subject manager
type Manager struct {
	ent                 *db.Client
	cache               *expirable.LRU[string, struct{}]
	cacheReloadInterval time.Duration
	cacheReloadTimeout  time.Duration
	cachePrefillCount   int
	cacheSize           int
	logger              *slog.Logger
	paginationSize      int
	lastRefreshAt       *time.Time
}

// EventBatchedIngestHandlerFactory returns a handler for batched ingest events
// The handler ensures that the subjects in the ingested events exist in the database.
// This handler is used in the watermill event router.
func (m *Manager) EventBatchedIngestHandlerFactory() cqrs.GroupEventHandler {
	return cqrs.NewGroupEventHandler(func(ctx context.Context, event *ingestevents.EventBatchedIngest) error {
		if event == nil {
			return errors.New("nil batched ingest event")
		}

		// Collect the subjects that need to be upserted
		params := []*SubjectRef{
			{
				Namespace: event.Namespace.ID,
				Key:       event.SubjectKey,
			},
		}

		// Ensure the subjects exist in the database
		return m.Ensure(ctx, params...)
	})
}

// Ensure creates subjects when they do not exist in the cache
func (m *Manager) Ensure(ctx context.Context, params ...*SubjectRef) error {
	var upserts []*SubjectRef

	// Collect the subjects that need to be upserted
	for _, param := range params {
		ok := m.getFromCache(param.Namespace, param.Key)
		if ok {
			continue
		}

		upserts = append(upserts, param)
	}

	// Do nothing if there are no subjects to upsert
	if len(upserts) == 0 {
		return nil
	}

	// Upsert the subjects
	var subjectCreates []*db.SubjectCreate

	for _, param := range upserts {
		subjectCreate := m.ent.Subject.Create().SetKey(param.Key).SetNamespace(param.Namespace)
		subjectCreates = append(subjectCreates, subjectCreate)
	}

	err := m.ent.Subject.
		CreateBulk(subjectCreates...).
		// Upsert
		OnConflict().
		DoNothing().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert subjects: %w", err)
	}

	// Add the subjects to the cache
	for _, param := range upserts {
		m.addToCache(param.Namespace, param.Key)
	}

	m.logger.DebugContext(ctx, "upserted subjects", "count", len(upserts))

	return nil
}

// scheduleCacheReload schedules a reload of the subjects
func (m *Manager) scheduleCacheReload() {
	m.logger.DebugContext(context.Background(), "scheduling cache reload")

	// Add a random jitter to the reload interval so that the subjects are not refreshed at the same time
	reloadInterval := m.cacheReloadInterval + time.Duration(rand.Intn(int(m.cacheReloadInterval)/2))

	time.AfterFunc(reloadInterval, func() {
		m.reloadCache(reloadDifferential)
	})
}

type reloadMode int

const (
	// reloadPrefill reloads the cache for up to PrefillMaxItems subjects
	reloadPrefill reloadMode = iota
	// reloadDifferential reloads the cache for the subjects that were created after the last refresh
	reloadDifferential
)

// reloadCache reloads the cache of subjects
func (m *Manager) reloadCache(mode reloadMode) {
	ctx := context.Background()

	// Set a timeout for the cache reload
	ctx, cancel := context.WithTimeout(ctx, m.cacheReloadTimeout)
	defer cancel()

	// Schedule the next reload
	defer m.scheduleCacheReload()

	// Get all subjects
	limit := m.paginationSize
	offset := 0

	var (
		subjectEntities []*db.Subject
		err             error
	)

	refreshStartedAt := time.Now()

	var maxFetchCount int
	if mode == reloadPrefill {
		maxFetchCount = m.cachePrefillCount
	} else {
		maxFetchCount = m.cacheSize
	}

	// Get all subjects via pagination
	for {
		query := m.ent.Subject.Query()

		if mode == reloadDifferential {
			if m.lastRefreshAt != nil {
				query = query.Where(dbsubject.CreatedAtGT(*m.lastRefreshAt))
			} else {
				m.logger.Warn("no lastRefreshAt is set, fetching subjects from the last 24 hours")
				query = query.Where(dbsubject.CreatedAtGT(time.Now().Add(-24 * time.Hour)))
			}
		}

		subjectEntities, err = query.
			Order(dbsubject.ByCreatedAt(sql.OrderDesc())).
			Offset(offset).
			Limit(limit).
			All(ctx)
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to list subjects", "error", err, "offset", offset, "limit", limit)
			return
		}

		// Add the subjects to the cache
		for _, subjectEntity := range subjectEntities {
			m.addToCache(subjectEntity.Namespace, subjectEntity.Key)
		}

		// Stop pagination if there are no more subjects
		if len(subjectEntities) < limit {
			break
		}

		offset += len(subjectEntities)

		// We can get marginally more items for prefetch, but it does not matter much
		if maxFetchCount > 0 && offset >= maxFetchCount {
			break
		}
	}

	if offset >= maxFetchCount && mode == reloadDifferential {
		m.logger.Warn("fetched more subjects for differential reload than the cache size, expect heavy cache thrashing", "fetched_count", offset, "cache_size", m.cacheSize)
	}

	m.lastRefreshAt = &refreshStartedAt
}

// AddToCache adds a subject to the cache
func (m *Manager) addToCache(namespace, key string) {
	m.cache.Add(getCacheKey(namespace, key), struct{}{})
}

// GetFromCache gets a subject from the cache if it exists
func (m *Manager) getFromCache(namespace, key string) bool {
	_, ok := m.cache.Get(getCacheKey(namespace, key))

	return ok
}

// getCacheKey returns the cache key
func getCacheKey(ns, key string) string {
	return fmt.Sprintf("%s::%s", ns, key)
}
