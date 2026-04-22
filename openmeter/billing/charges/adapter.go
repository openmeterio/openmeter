package charges

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ChargesSearchAdapter

	entutils.TxCreator
}

type ChargesSearchAdapter interface {
	GetByIDs(ctx context.Context, input GetByIDsInput) (ChargeSearchItems, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[ChargeSearchItem], error)
	ListCustomersToAdvance(ctx context.Context, input ListCustomersToAdvanceInput) (pagination.Result[customer.CustomerID], error)
}

type ChargeSearchItem struct {
	ID         meta.ChargeID
	Type       meta.ChargeType
	CustomerID string
}

func (c *ChargeSearchItem) Validate() error {
	var errs []error
	if err := c.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id: %w", err))
	}

	if err := c.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if c.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ChargeSearchItems []ChargeSearchItem

func (c ChargeSearchItems) Validate() error {
	var errs []error
	for idx, item := range c {
		if err := item.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("item[%d]: %w", idx, err))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
