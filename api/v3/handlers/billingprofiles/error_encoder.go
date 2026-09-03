package billingprofiles

import (
	"github.com/openmeterio/openmeter/api/v3/handlers/billingerrors"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

func errorEncoder() encoder.ErrorEncoder {
	return billingerrors.ErrorEncoder()
}
