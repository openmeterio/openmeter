package charges

import (
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// CustomerCharge is the API-facing view of a charge returned by the
// customer-charge facade: the domain charge plus the side-loaded entities and
// the resolved realization history the wire contract needs.
//
// It lives on the charges package rather than the concrete charge types
// because openmeter/subscription itself depends on the concrete charge
// packages, so any API-facing entity carried by them would create an import
// cycle.
type CustomerCharge struct {
	Charge // domain union, unchanged

	// Customer, Feature, and Subscription are nil unless their expand was
	// applied (and the referenced entity exists); API converters fall back to
	// the id references carried by the charge itself.
	Customer     *customer.Customer
	Feature      *feature.Feature
	Subscription *subscription.Subscription

	// Exactly the slice matching Charge.Type() is populated.
	FlatFeeRealizations    []CustomerChargeFlatFeeRealization
	UsageBasedRealizations []CustomerChargeUsageBasedRealization
}

// CustomerChargeFlatFeeRealization is a presentation-ready view of one entry
// of a flat-fee charge's realization history: a booked run with its own
// persisted service period, or — when Run is nil — the outstanding projection
// spanning the whole service period (flat fees are never partially invoiced,
// so a charge with no live run resolves to this single entry). Flat fee runs
// persist their own service period, so unlike usage-based charges no
// derivation happens between neighbors, and there is no quantity: a flat
// fee's amount does not decompose per realization entry.
type CustomerChargeFlatFeeRealization struct {
	// Run is the booked run; nil marks the outstanding-tail projection.
	Run *flatfee.RealizationRun
	// ServicePeriod is the period this entry accounts for.
	ServicePeriod timeutil.ClosedPeriod
	// Voided marks runs whose billing effect was undone. Voided entries are
	// audit history: they do not count toward the covered service period.
	Voided bool
	// Invoice is the header (lines stripped) of the invoice the run realized
	// into, attached by the customer-charge API facade under the
	// realization-invoice expand; nil otherwise (converters fall back to the
	// run's invoice ID).
	Invoice *billing.StandardInvoice
}

// CustomerChargeUsageBasedRealization is a presentation-ready view of one
// entry of a usage-based charge's realization history: a booked run with its
// derived service period and de-cumulated quantity, or the outstanding-tail
// projection when Run is nil.
//
// Runs only persist the end of their service period and a cumulative metered
// quantity, so both the period start and the run's own consumption are
// derived from the previously created non-voided run.
type CustomerChargeUsageBasedRealization struct {
	// Run is the booked run; nil marks the outstanding-tail projection.
	Run *usagebased.RealizationRun
	// ServicePeriod is the derived period this entry accounts for.
	ServicePeriod timeutil.ClosedPeriod
	// Quantity is the entry's own metered consumption, reported as a signed
	// delta: booked entries go negative after a downward usage correction or
	// when a voided run snapshotted behind its live neighbors, stating the
	// correction frankly instead of fabricating a zero. Only the outstanding
	// projection is floored at zero (per the wire contract); it is zero
	// unless a live metering read was supplied through the realtime usage
	// expand.
	Quantity alpacadecimal.Decimal
	// Voided marks runs whose billing effect was undone (deleted runs and
	// unsupported-credit-note runs). Voided entries are audit history: they
	// never advance the derived periods or quantities of the surrounding live
	// entries.
	Voided bool
	// Invoice is the header (lines stripped) of the invoice the run realized
	// into, attached by the customer-charge API facade under the
	// realization-invoice expand; nil otherwise (converters fall back to the
	// run's invoice ID).
	Invoice *billing.StandardInvoice
}
