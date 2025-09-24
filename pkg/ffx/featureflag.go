// Package ffx provides a simple feature flag service.
package ffx

import "context"

type Feature string

type Service interface {
	IsFeatureEnabled(ctx context.Context, feature Feature) (bool, error)
}

type AccessConfig map[Feature]bool

func (c AccessConfig) Merge(other AccessConfig) AccessConfig {
	for feature, value := range other {
		c[feature] = value
	}

	return c
}
