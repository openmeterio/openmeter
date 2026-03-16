package ledgerv2

import (
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/samber/mo"
)

// SubAccountDimensions is a set of all known dimensins for a generic sub-account
type SubAccountDimensions struct {
	Currency       mo.Option[currencyx.Code]
	TaxCode        mo.Option[string]
	CreditPriority mo.Option[int]
	Feature        mo.Option[[]string]
}

func (d SubAccountDimensions) Validate() error {
	// TODO
	return nil
}

func (d SubAccountDimensions) Dimensions() SubAccountDimensions {
	return d
}
