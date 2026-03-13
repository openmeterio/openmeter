package charges

import (
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	entutils.TxCreator
}
