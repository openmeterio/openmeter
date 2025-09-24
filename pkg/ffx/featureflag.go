// Package ffx provides a simple feature flag service.
package ffx

import "context"

type Feature string

type Service interface {
	IsFeatureEnabled(ctx context.Context, feature Feature) (bool, error)
}

type AccessConfig map[Feature]bool
