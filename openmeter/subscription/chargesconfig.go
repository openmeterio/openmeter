package subscription

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

// ChargesConfig says when a subscription is billed through charges instead of legacy invoice lines.
// It has to agree with how subscription sync routes items, see ResolveDefaultCollection in
// openmeter/billing/worker/subscriptionsync/service/reconciler.
type ChargesConfig struct {
	// Enabled is the deployment level switch for credits (and with it, charges).
	Enabled bool
	// EnableCreditThenInvoice moves credit_then_invoice subscriptions to charges too.
	// Without it only credit_only subscriptions use them.
	EnableCreditThenInvoice bool
	// FeatureGate decides per namespace whether credits are available.
	FeatureGate *featuregate.FeatureGateChecker
}

func (c ChargesConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// A checker without a gate reports credits as enabled for every namespace instead of failing,
	// so reject it here like subscription sync does.
	if err := c.FeatureGate.Validate(); err != nil {
		return fmt.Errorf("charges: %w", err)
	}

	return nil
}

func (c ChargesConfig) IsChargeBacked(namespace string, settlementMode productcatalog.SettlementMode) (bool, error) {
	if !c.Enabled {
		return false, nil
	}

	creditsEnabled, err := c.FeatureGate.Enabled(namespace, c.FeatureGate.Flags.Credits())
	if err != nil {
		return false, err
	}

	if !creditsEnabled {
		return false, nil
	}

	return settlementMode != productcatalog.CreditThenInvoiceSettlementMode || c.EnableCreditThenInvoice, nil
}
