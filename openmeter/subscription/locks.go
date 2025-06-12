package subscription

import "github.com/openmeterio/openmeter/pkg/framework/lockr"

func GetCustomerLock(customerId string) (lockr.Key, error) {
	return lockr.NewKey("customer", customerId, "subscription")
}
