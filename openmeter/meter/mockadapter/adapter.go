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
}

func (c *adapter) init() {
	c.initOnce.Do(func() {
		if c.meters == nil {
			c.meters = make([]meter.Meter, 0)
		}
	})
}

type TestAdapter = adapter
