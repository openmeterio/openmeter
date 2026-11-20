package subscription

import "github.com/openmeterio/openmeter/pkg/ffx"

const MultiSubscriptionEnabledFF = ffx.Feature("multi-subscription-enabled")

// BillingStartsAtEnabledFF lets a namespace create subscriptions whose billing starts later than the subscription itself.
const BillingStartsAtEnabledFF = ffx.Feature("billing-starts-at-enabled")
