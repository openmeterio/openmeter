package meta

import (
	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Patch interface {
	models.Validator

	Trigger() stateless.Trigger
	// Note: trigger params is any as stateless only support this as an input argument
	TriggerParams() any
}

type TriggerPatchResult[T any] struct {
	Charge         *T
	InvoicePatches []invoiceupdater.Patch
}
