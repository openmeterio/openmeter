package splitlinegroup

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	timeutil "github.com/openmeterio/openmeter/pkg/timeutil"
)

type SplitLineGroupMutableFields struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    models.Metadata `json:"metadata,omitempty"`

	ServicePeriod timeutil.ClosedPeriod `json:"period"`

	RatecardDiscounts billing.Discounts `json:"ratecardDiscounts"`
}

func (i SplitLineGroupMutableFields) ValidateForPrice(price *productcatalog.Price) error {
	var errs []error

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.RatecardDiscounts.ValidateForPrice(price); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (i SplitLineGroupMutableFields) Clone() SplitLineGroupMutableFields {
	clone := i
	clone.RatecardDiscounts = i.RatecardDiscounts.Clone()

	return clone
}

type SplitLineGroupCreate struct {
	Namespace string `json:"namespace"`

	SplitLineGroupMutableFields `json:",inline"`

	Price             *productcatalog.Price          `json:"price"`
	FeatureKey        *string                        `json:"featureKey,omitempty"`
	Subscription      *billing.SubscriptionReference `json:"subscription,omitempty"`
	Currency          currencyx.Code                 `json:"currency"`
	UniqueReferenceID *string                        `json:"childUniqueReferenceId,omitempty"`
}

func (i SplitLineGroupCreate) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(i.Price); err != nil {
		errs = append(errs, err)
	}

	if i.Price == nil {
		errs = append(errs, errors.New("price is required"))
	} else {
		if err := i.Price.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, errors.New("unique reference id is required"))
	}

	return errors.Join(errs...)
}

type SplitLineGroupUpdate struct {
	models.NamespacedID `json:",inline"`

	SplitLineGroupMutableFields `json:",inline"`
}

func (i SplitLineGroupUpdate) ValidateWithPrice(price *productcatalog.Price) error {
	var errs []error

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(price); err != nil {
		errs = append(errs, err)
	}

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type SplitLineGroup struct {
	models.ManagedModel         `json:",inline"`
	models.NamespacedID         `json:",inline"`
	SplitLineGroupMutableFields `json:",inline"`

	Price             *productcatalog.Price          `json:"price"`
	FeatureKey        *string                        `json:"featureKey,omitempty"`
	Subscription      *billing.SubscriptionReference `json:"subscription,omitempty"`
	Currency          currencyx.Code                 `json:"currency"`
	UniqueReferenceID *string                        `json:"childUniqueReferenceId,omitempty"`
}

func (i SplitLineGroup) Validate() error {
	var errs []error

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(i.Price); err != nil {
		errs = append(errs, err)
	}

	if i.Price != nil {
		if err := i.Price.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	return errors.Join(errs...)
}

func (i SplitLineGroup) ToUpdate() SplitLineGroupUpdate {
	return SplitLineGroupUpdate{
		NamespacedID:                i.NamespacedID,
		SplitLineGroupMutableFields: i.SplitLineGroupMutableFields.Clone(),
	}
}

func (i SplitLineGroup) Clone() SplitLineGroup {
	return SplitLineGroup{
		ManagedModel:                i.ManagedModel,
		NamespacedID:                i.NamespacedID,
		SplitLineGroupMutableFields: i.SplitLineGroupMutableFields.Clone(),
		Price:                       i.Price,
		FeatureKey:                  i.FeatureKey,
		Subscription:                i.Subscription,
		Currency:                    i.Currency,
		UniqueReferenceID:           i.UniqueReferenceID,
	}
}

type SplitLineHierarchy struct {
	Group SplitLineGroup

	StandardLines StandardLines
	GatheringLine *GatheringLine
}

func (h SplitLineHierarchy) Clone() (SplitLineHierarchy, error) {
	standardLines, err := lo.MapErr(h.StandardLines, func(line StandardLine, _ int) (StandardLine, error) {
		return line.Clone()
	})
	if err != nil {
		return SplitLineHierarchy{}, fmt.Errorf("cloning standard lines: %w", err)
	}

	gatheringLine, err := h.GatheringLine.CloneOrNil()
	if err != nil {
		return SplitLineHierarchy{}, fmt.Errorf("cloning gathering line: %w", err)
	}

	return SplitLineHierarchy{
		Group:         h.Group.Clone(),
		StandardLines: standardLines,
		GatheringLine: gatheringLine,
	}, nil
}

func (h SplitLineHierarchy) Lines() []LineHeaderAccessor {
	lines := lo.Map(h.StandardLines, func(line StandardLine, _ int) LineHeaderAccessor {
		return line
	})

	if h.GatheringLine != nil {
		lines = append(lines, *h.GatheringLine)
	}
	return lines
}

type SumNetAmountInput struct {
	PeriodEndLTE   time.Time
	IncludeCharges bool
}

// SumNetAmount returns the sum of the net amount (pre-tax) of the progressive billed line and its children
// containing the values for all lines whose period's end is <= in.UpTo and are not deleted or not part of
// an invoice that has been deleted.
// As gathering lines do not represent any kind of actual charge, they are not included in the sum.
func (h *SplitLineHierarchy) SumNetAmount(in SumNetAmountInput) (alpacadecimal.Decimal, error) {
	netAmount := alpacadecimal.Zero

	err := h.ForEachStandardLine(ForEachStandardLineInput{
		PeriodEndLTE: in.PeriodEndLTE,
		Callback: func(line StandardLine) error {
			netAmount = netAmount.Add(line.Totals.Amount)

			if in.IncludeCharges {
				netAmount = netAmount.Add(line.Totals.ChargesTotal)
			}

			return nil
		},
	})
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return netAmount, nil
}

type ForEachStandardLineInput struct {
	PeriodEndLTE time.Time
	Callback     func(line StandardLine) error
}

func (h *SplitLineHierarchy) ForEachStandardLine(in ForEachStandardLineInput) error {
	for _, line := range h.StandardLines {
		// The line is not in scope
		if !in.PeriodEndLTE.IsZero() && line.ServicePeriod.To.After(in.PeriodEndLTE) {
			continue
		}

		if line.DeletedAt != nil {
			continue
		}

		if line.Invoice.DeletedAt != nil {
			continue
		}

		if err := in.Callback(line); err != nil {
			return err
		}
	}

	return nil
}

// Adapter
type (
	// TODO: Remove type aliases
	CreateSplitLineGroupAdapterInput = SplitLineGroupCreate
	UpdateSplitLineGroupInput        = SplitLineGroupUpdate
	DeleteSplitLineGroupInput        = models.NamespacedID
	GetSplitLineGroupInput           = models.NamespacedID
)

type GetSplitLineGroupHeadersInput struct {
	Namespace         string
	SplitLineGroupIDs []string
}

type SplitLineGroupHeaders = []SplitLineGroup

func (i GetSplitLineGroupHeadersInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	return errors.Join(errs...)
}
