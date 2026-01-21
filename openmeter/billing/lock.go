package billing

import "github.com/openmeterio/openmeter/openmeter/customer"

type (
	LockCustomerForUpdateAdapterInput = customer.CustomerID
	UpsertCustomerLockAdapterInput    = LockCustomerForUpdateAdapterInput
)
