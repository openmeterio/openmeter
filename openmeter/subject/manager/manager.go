package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
)

// Config is the configuration for the subject manager
type Config struct {
	CacheReloadInterval time.Duration
	CacheReloadTimeout  time.Duration
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
		logger:              config.Logger.WithGroup("subject-manager"),
		mux:                 sync.RWMutex{},
		paginationSize:      config.PaginationSize,
	}

	// Initialize cache and schedule next reload
	manager.reloadCache()

	return manager, nil
}

// Manager is a subject manager
type Manager struct {
	ent                 *db.Client
	cache               *expirable.LRU[string, struct{}]
	cacheReloadInterval time.Duration
	cacheReloadTimeout  time.Duration
	logger              *slog.Logger
	mux                 sync.RWMutex
	paginationSize      int
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
	time.AfterFunc(m.cacheReloadInterval, m.reloadCache)
}

// reloadCache reloads the cache of subjects
func (m *Manager) reloadCache() {
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

	// We pre-build the next cache to avoid having an empty cache during the reload
	nextCache := make(map[string]struct{})

	// Get all subjects via pagination
	for {
		subjectEntities, err = m.ent.Subject.
			Query().
			Offset(offset).
			Limit(limit).
			All(ctx)
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to list subjects", "error", err, "offset", offset, "limit", limit)
			return
		}

		// Add the subjects to the cache
		for _, subjectEntity := range subjectEntities {
			key := getCacheKey(subjectEntity.Namespace, subjectEntity.Key)
			nextCache[key] = struct{}{}
		}

		// Stop pagination if there are no more subjects
		if len(subjectEntities) < limit {
			break
		}

		offset += limit
	}

	// Replace cache
	m.mux.Lock()
	defer m.mux.Unlock()

	m.cache.Purge()

	for key := range nextCache {
		m.cache.Add(key, struct{}{})
	}
}

// AddToCache adds a subject to the cache
func (m *Manager) addToCache(namespace, key string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.cache.Add(getCacheKey(namespace, key), struct{}{})
}

// GetFromCache gets a subject from the cache if it exists
func (m *Manager) getFromCache(namespace, key string) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()

	_, ok := m.cache.Get(getCacheKey(namespace, key))

	return ok
}

// getCacheKey returns the cache key
func getCacheKey(ns, key string) string {
	return fmt.Sprintf("%s::%s", ns, key)
}
