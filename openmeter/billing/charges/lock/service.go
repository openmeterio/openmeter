package lock

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type LockChargeInput = meta.ChargeID

func NewChargeKey(chargeID meta.ChargeID) (lockr.Key, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("charge ID: %w", err)
	}

	return lockr.NewKey("namespace", chargeID.Namespace, "charge", chargeID.ID)
}
