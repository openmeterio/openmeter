package charges

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Service interface {
	ChargeService

	// Facade interfaces provide convinience helpers for the API layer.
	CreditPurchaseFacadeService
}

type ChargeService interface {
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) (Charges, error)
	Create(ctx context.Context, input CreateInput) (Charges, error)

	AdvanceCharges(ctx context.Context, input AdvanceChargesInput) (Charges, error)
	ListCustomersToAdvance(ctx context.Context, input ListCustomersToAdvanceInput) (pagination.Result[customer.CustomerID], error)
	// ApplyPatches currently returns no affected-charge payload. If exact post-apply
	// results are needed, shrink/extend must first be implemented properly instead of
	// going through the temporary delete+create remap.
	ApplyPatches(ctx context.Context, input ApplyPatchesInput) error
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)
}

type CreditPurchaseFacadeService interface {
	HandleCreditPurchaseExternalPaymentStateTransition(ctx context.Context, input HandleCreditPurchaseExternalPaymentStateTransitionInput) (creditpurchase.Charge, error)
}

type CreateInput struct {
	Namespace string
	Intents   ChargeIntents
}

func (i CreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Intents.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intents: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetByIDInput struct {
	ChargeID meta.ChargeID
	Expands  meta.Expands
}

func (i GetByIDInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetByIDsInput struct {
	Namespace string
	IDs       []string
	Expands   meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, id := range i.IDs {
		if id == "" {
			errs = append(errs, errors.New("id is required"))
		}
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type HandleCreditPurchaseExternalPaymentStateTransitionInput struct {
	ChargeID meta.ChargeID

	TargetPaymentState payment.Status
}

func (i HandleCreditPurchaseExternalPaymentStateTransitionInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.TargetPaymentState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target payment state: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AdvanceChargesInput struct {
	Customer customer.CustomerID
}

func (i AdvanceChargesInput) Validate() error {
	var errs []error
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListChargesInput struct {
	pagination.Page

	Namespace       string
	CustomerIDs     []string
	SubscriptionIDs []string
	ChargeTypes     []meta.ChargeType
	StatusIn        []meta.ChargeStatus
	StatusNotIn     []meta.ChargeStatus
	IncludeDeleted  bool

	// OrderBy is the field to sort by. Supported values: id, created_at,
	// service_period.from, billing_period.from.
	// Defaults to created_at when empty.
	OrderBy string
	Order   sortx.Order

	Expands meta.Expands
}

func (i ListChargesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, customerID := range i.CustomerIDs {
		if customerID == "" {
			errs = append(errs, errors.New("customer id is required"))
		}
	}

	for _, subscriptionID := range i.SubscriptionIDs {
		if subscriptionID == "" {
			errs = append(errs, errors.New("subscription id is required"))
		}
	}

	for _, chargeType := range i.ChargeTypes {
		if err := chargeType.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("charge type: %w", err))
		}
	}

	for _, status := range i.StatusIn {
		if err := status.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("status: %w", err))
		}
	}

	for _, status := range i.StatusNotIn {
		if err := status.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("status: %w", err))
		}
	}

	if len(i.StatusIn) > 0 && len(i.StatusNotIn) > 0 {
		errs = append(errs, errors.New("status_in and status_not_in cannot be set at the same time"))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListCustomersToAdvanceInput struct {
	pagination.Page

	Namespaces      []string
	AdvanceAfterLTE time.Time
}

func (i ListCustomersToAdvanceInput) Validate() error {
	if i.AdvanceAfterLTE.IsZero() {
		return models.NewGenericValidationError(errors.New("advance_after_lte is required"))
	}

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("page: %w", err))
		}
	}

	return nil
}
