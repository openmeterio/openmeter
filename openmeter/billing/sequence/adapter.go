package sequence

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	entutils.TxCreator

	NextSequenceNumber(ctx context.Context, input NextSequenceNumberInput) (alpacadecimal.Decimal, error)
}
