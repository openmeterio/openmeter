package charges

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ChargesSearchAdapter

	entutils.TxCreator
}

type ChargesSearchAdapter interface {
	GetByIDs(ctx context.Context, input GetByIDsInput) (ChargeSearchItems, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[ChargeSearchItem], error)
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
	return errors.Join(errs...)
}

type ChargeSearchItems []ChargeSearchItem

func (c ChargeSearchItems) Validate() error {
	var errs []error
	for idx, item := range c {
		if err := item.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("item[%d]: %w", idx, err))
		}
	}
	return errors.Join(errs...)
}

type GetTypesByIDsInput struct {
	Namespace string
	IDs       []string
}

func (i GetTypesByIDsInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, id := range i.IDs {
		if id == "" {
			errs = append(errs, errors.New("id is required"))
		}
	}

	return errors.Join(errs...)
}
