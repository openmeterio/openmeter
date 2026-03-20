package meta

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/expand"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargeID models.NamespacedID

func (i ChargeID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type ChargeType string

const (
	ChargeTypeFlatFee        ChargeType = "flat_fee"
	ChargeTypeUsageBased     ChargeType = "usage_based"
	ChargeTypeCreditPurchase ChargeType = "credit_purchase"
)

func (t ChargeType) Values() []string {
	return []string{
		string(ChargeTypeFlatFee),
		string(ChargeTypeUsageBased),
		string(ChargeTypeCreditPurchase),
	}
}

func (t ChargeType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return fmt.Errorf("invalid charge type: %s", t)
	}

	return nil
}

type Expand string

const (
	ExpandRealizations Expand = "realizations"
)

func (e Expand) Values() []Expand {
	return []Expand{
		ExpandRealizations,
	}
}

var ExpandNone Expands = nil

type Expands = expand.Expand[Expand]

type ChargeAccessor interface {
	GetChargeID() ChargeID
	ErrorAttributes() models.Attributes
}

type ChargeStatus string

const (
	// ChargeStatusCreated is the status of a charge that is created and is not yet active.
	ChargeStatusCreated ChargeStatus = "created"
	// ChargeStatusActive is the status of a charge that is active and is not yet fully settled for the service period.
	ChargeStatusActive ChargeStatus = "active"
	// ChargeStatusSettled is the status of a charge that is settled and is fully settled for the service period. The charge might receive additional
	// late events in the future.
	ChargeStatusSettled ChargeStatus = "settled"
	// ChargeStatusFinal is the status of a charge that is final and is fully settled for the service period. The charge will not receive any additional
	// late events in the future.
	ChargeStatusFinal ChargeStatus = "final"
)

func (s ChargeStatus) Values() []string {
	return []string{
		string(ChargeStatusCreated),
		string(ChargeStatusActive),
		string(ChargeStatusSettled),
		string(ChargeStatusFinal),
	}
}

func (s ChargeStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid charge status: %s", s)
	}

	return nil
}

type Charge struct {
	ManagedResource

	Intent       Intent
	Type         ChargeType
	Status       ChargeStatus
	AdvanceAfter *time.Time
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if err := c.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed resource: %w", err))
	}

	return errors.Join(errs...)
}

type Charges []Charge

func (c Charges) Validate() error {
	var errs []error

	for i, ch := range c {
		if err := ch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("charge [%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}
