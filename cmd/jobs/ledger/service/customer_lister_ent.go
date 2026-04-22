package service

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type EntCustomerLister struct {
	entClient *entdb.Client
}

func NewEntCustomerLister(entClient *entdb.Client) *EntCustomerLister {
	return &EntCustomerLister{
		entClient: entClient,
	}
}

func (l *EntCustomerLister) ListCustomers(ctx context.Context, input ListCustomersInput) (ListCustomersResult, error) {
	if err := input.Validate(); err != nil {
		return ListCustomersResult{}, fmt.Errorf("invalid list customers input: %w", err)
	}

	query := l.entClient.Customer.Query().
		Where(customerdb.Namespace(input.Namespace)).
		Limit(input.PageSize)

	if !input.IncludeDeleted {
		now := clock.Now().UTC()
		query = query.Where(customerdb.Or(
			customerdb.DeletedAtIsNil(),
			customerdb.DeletedAtGTE(now),
		))
	}

	if input.CreatedBefore != nil {
		query = query.Where(customerdb.CreatedAtLT(input.CreatedBefore.UTC()))
	}

	paged, err := query.Cursor(ctx, input.Cursor)
	if err != nil {
		return ListCustomersResult{}, fmt.Errorf("query customers: %w", err)
	}

	items := make([]CustomerListItem, 0, len(paged.Items))
	for _, item := range paged.Items {
		if item == nil {
			continue
		}

		items = append(items, CustomerListItem{
			ID:        item.ID,
			CreatedAt: item.CreatedAt.UTC(),
		})
	}

	return ListCustomersResult{
		Items:      items,
		NextCursor: paged.NextCursor,
	}, nil
}
