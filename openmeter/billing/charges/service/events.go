package service

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func (s *service) publishCustomerChargesChanged(ctx context.Context, owners []customer.CustomerID) {
	for _, owner := range lo.Uniq(owners) {
		s.customerChargesChangedPublisher(ctx, charges.CustomerChargesChangedEvent{Namespace: owner.Namespace, CustomerID: owner.ID})
	}
}
