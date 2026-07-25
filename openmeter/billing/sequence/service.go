package sequence

import "context"

type Service interface {
	GenerateInvoiceSequenceNumber(ctx context.Context, in GenerationInput, def Definition) (string, error)
}
