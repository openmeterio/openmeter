package appsandbox

import "github.com/openmeterio/openmeter/openmeter/billing"

var ErrSimulatedPaymentFailure = billing.NewValidationError("simulated_payment_failure", "simulated payment failure")
