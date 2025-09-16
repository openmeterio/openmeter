package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ meter.Service = (*Service)(nil)

type Service struct {
	adapter *adapter.Adapter
}

func New(
	adapter *adapter.Adapter,
) *Service {
	return &Service{
		adapter: adapter,
	}
}

// ListMeters lists meters
func (s *Service) ListMeters(ctx context.Context, input meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return s.adapter.ListMeters(ctx, input)
}

// GetMeterByIDOrSlug gets a meter
func (s *Service) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	return s.adapter.GetMeterByIDOrSlug(ctx, input)
}
