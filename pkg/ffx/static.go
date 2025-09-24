package ffx

import (
	"context"
	"fmt"
)

type staticService struct {
	config AccessConfig
}

var _ Service = &staticService{}

func (s *staticService) IsFeatureEnabled(ctx context.Context, feature Feature) (bool, error) {
	acc, ok := s.config[feature]
	if !ok {
		return false, fmt.Errorf("feature %s not found", feature)
	}

	return acc, nil
}

func NewStaticService(config AccessConfig) Service {
	return &staticService{
		config: config,
	}
}
