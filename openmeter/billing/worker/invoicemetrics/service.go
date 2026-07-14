package invoicemetrics

import "context"

type Service interface {
	Start(ctx context.Context) error
	Stop()
}
