package models

import (
	"context"
	"time"
)

type CadencedResourceRepo[T Cadenced] interface {
	SetEndOfCadence(ctx context.Context, id NamespacedID, at *time.Time) (*T, error)
}
