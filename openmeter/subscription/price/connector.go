package price

import "context"

type Connector interface {
	Create(ctx context.Context, input CreateInput) (*Price, error)
}
