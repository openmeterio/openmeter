package charges

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// CustomerCharge is the API-facing view of a charge: the domain charge plus
// its expanded entities and resolved realization history.
type CustomerCharge struct {
	Charge

	// Customer, Feature, and Subscription are nil unless expanded and the
	// referenced entity exists; converters fall back to the charge's own ID
	// references.
	Customer     *customer.Customer
	Feature      *feature.Feature
	Subscription *subscription.Subscription

	// Exactly the slice matching Charge.Type() is populated.
	FlatFeeRealizations    []CustomerChargeFlatFeeRealization
	UsageBasedRealizations []CustomerChargeUsageBasedRealization
}

// ForSpecificChargeInput holds one handler per charge type. A nil handler
// marks the operation as unsupported for that type.
type ForSpecificChargeInput[T any] struct {
	FlatFee        func(flatfee.Charge) (T, error)
	UsageBased     func(usagebased.Charge) (T, error)
	CreditPurchase func(creditpurchase.Charge) (T, error)
}

// ForSpecificCharge dispatches to the handler matching the charge type.
func ForSpecificCharge[T any](c CustomerCharge, in ForSpecificChargeInput[T]) (T, error) {
	switch c.t {
	case meta.ChargeTypeFlatFee:
		if in.FlatFee == nil {
			return lo.Empty[T](), fmt.Errorf("unsupported for flat fee charge")
		}

		ff, err := c.AsFlatFeeCharge()
		if err != nil {
			return lo.Empty[T](), err
		}

		return in.FlatFee(ff)
	case meta.ChargeTypeUsageBased:
		if in.UsageBased == nil {
			return lo.Empty[T](), fmt.Errorf("unsupported for usage based charge")
		}

		ub, err := c.AsUsageBasedCharge()
		if err != nil {
			return lo.Empty[T](), err
		}

		return in.UsageBased(ub)
	case meta.ChargeTypeCreditPurchase:
		if in.CreditPurchase == nil {
			return lo.Empty[T](), fmt.Errorf("unsupported for credit purchase charge")
		}

		cp, err := c.AsCreditPurchaseCharge()
		if err != nil {
			return lo.Empty[T](), err
		}

		return in.CreditPurchase(cp)
	}

	return lo.Empty[T](), fmt.Errorf("invalid charge type: %s", c.t)
}

// GetResolvedCostBasis returns the fiat conversion rate exposed to clients. It
// is nil without a cost basis or before resolution. A dynamic cost basis is
// hidden until the service period starts: activation can resolve it earlier
// for in-advance charges, but the rate is only meaningful from the period it
// was resolved for.
func (c CustomerCharge) GetResolvedCostBasis() (*costbasis.State, error) {
	return ForSpecificCharge(c, ForSpecificChargeInput[*costbasis.State]{
		FlatFee: func(ff flatfee.Charge) (*costbasis.State, error) {
			return ff.State.ResolvedCostBasis, nil
		},
		UsageBased: func(ub usagebased.Charge) (*costbasis.State, error) {
			return ub.State.ResolvedCostBasis, nil
		},
	})
}

// CustomerChargeFlatFeeRealization is one entry of a flat-fee charge's
// realization history: a booked run, or the outstanding entry when Run is
// nil. A flat fee is never partially invoiced, so a charge without a live run
// carries a single outstanding entry covering the whole service period after
// its voided history. There is no quantity: a flat fee's amount does not
// split across entries.
type CustomerChargeFlatFeeRealization struct {
	// Run is the booked run; nil marks the outstanding entry.
	Run           *flatfee.RealizationRun
	ServicePeriod timeutil.ClosedPeriod
	Voided        bool
	Invoice       *billing.StandardInvoice
}

// CustomerChargeUsageBasedRealization is one entry of a usage-based charge's
// realization history: a booked run, or the outstanding entry when Run is
// nil. Runs persist only a cumulative quantity and a period end, so each
// entry's period start and quantity are derived from the previous live run.
type CustomerChargeUsageBasedRealization struct {
	// Run is the booked run; nil marks the outstanding entry.
	Run           *usagebased.RealizationRun
	ServicePeriod timeutil.ClosedPeriod
	// Quantity is the usage this entry realized since the previous live run.
	// It is signed, as a downward usage correction is a valid billing event.
	// The outstanding entry is never negative and is zero unless the realtime
	// usage expand supplied a live quantity.
	Quantity alpacadecimal.Decimal
	Voided   bool
	Invoice  *billing.StandardInvoice
}
