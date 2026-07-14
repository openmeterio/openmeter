package invoicemetrics

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	entutils.TxCreator

	CountOverdueInvoices(ctx context.Context, input CountOverdueInvoicesInput) (OverdueInvoiceCounts, error)
}
