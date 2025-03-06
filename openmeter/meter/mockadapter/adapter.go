package adapter

import (
	"fmt"
	"slices"
	"sync"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func New(meters []meter.Meter) (*adapter, error) {
	a := &adapter{}

	a.init()

	for idx, m := range meters {
		// Window size is deprecated, it's always minute
		meters[idx].WindowSize = meter.WindowSizeMinute

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

type TestAdapter = adapter
