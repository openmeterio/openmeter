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

// CustomerCharge is the API-facing view of a charge: the domain charge plus
// its expanded entities and resolved realization history.
type CustomerCharge struct {
	Charge

	// Customer, Feature, and Subscription are nil unless expanded and the
	// referenced entity exists; converters fall back to the charge's own ID
	// references.
	Customer     *customer.Customer
	Feature      *feature.Feature
	Subscription *subscription.SubscriptionView

	// Exactly the slice matching Charge.Type() is populated.
	FlatFeeRealizations    []CustomerChargeFlatFeeRealization
	UsageBasedRealizations []CustomerChargeUsageBasedRealization
}

// CustomerChargeFlatFeeRealization is one entry of a flat-fee charge's
// realization history: a booked run, or the outstanding entry when Run is
// nil. A flat fee is never partially invoiced, so a charge without a live run
// resolves to a single outstanding entry covering the whole service period.
// There is no quantity: a flat fee's amount does not split across entries.
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
