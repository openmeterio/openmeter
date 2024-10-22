package models

import (
	"context"
	"time"
)

type CadencedResourceRepo[T Cadenced] interface {
	EndCadence(ctx context.Context, id string, at time.Time) (*T, error)
}
