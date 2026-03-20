package adapter

import (
	"context"
	"fmt"
	"slices"
	"sync"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func New(meters []meter.Meter) (*adapter, error) {
	a := &adapter{}

	a.init()

	for _, m := range meters {
		if err := m.Validate(); err != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("failed to validate meter: %w", err),
			)
		}
	}

	a.meters = slices.Clone(meters)

	return a, nil
}

var _ meter.Service = (*adapter)(nil)

type adapter struct {
	meters   []meter.Meter
	initOnce sync.Once
	// dbClient is optionally set to sync meters to PG for FK constraints on features.meter_id.
	dbClient *entdb.Client
}

func (c *adapter) init() {
	c.initOnce.Do(func() {
		if c.meters == nil {
			c.meters = make([]meter.Meter, 0)
		}
	})
}

func NewManage(meters []meter.Meter) (meter.ManageService, error) {
	adapter, err := New(meters)
	if err != nil {
		return nil, err
	}

	return &manageAdapter{
		adapter: adapter,
		Service: adapter,
	}, nil
}

var _ meter.ManageService = (*manageAdapter)(nil)

type manageAdapter struct {
	adapter *adapter

	meter.Service
}

// SetDBClient sets the ent DB client so that meter mutations are also persisted to PG.
// This ensures FK constraints on features.meter_id are satisfied in tests.
// It also syncs any existing in-memory meters to PG.
func (c *adapter) SetDBClient(client *entdb.Client) error {
	c.dbClient = client

	// Sync existing meters to PG.
	if len(c.meters) > 0 {
		if err := c.ReplaceMeters(context.Background(), c.meters); err != nil {
			c.dbClient = nil
			return fmt.Errorf("failed to sync meters to PG: %w", err)
		}
	}

	return nil
}

type TestAdapter = adapter
