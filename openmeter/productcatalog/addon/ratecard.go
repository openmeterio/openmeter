package addon

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator                      = (*RateCardManagedFields)(nil)
	_ models.Equaler[RateCardManagedFields] = (*RateCardManagedFields)(nil)
)

type RateCardManagedFields struct {
	models.ManagedModel
	models.NamespacedID

	// AddonID defines the Addon the RateCard assigned to.
	AddonID string `json:"addonId"`
}

func (m RateCardManagedFields) Equal(v RateCardManagedFields) bool {
	if m.Namespace != v.Namespace {
		return false
	}

	if m.ID != v.ID {
		return false
	}

	return m.AddonID == v.AddonID
}

func (m RateCardManagedFields) Validate() error {
	var errs []error

	if m.Namespace == "" {
		errs = append(errs, errors.New("namespace must not be empty"))
	}

	if m.ID == "" {
		errs = append(errs, errors.New("id must not be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ManagedRateCard interface {
	ManagedFields() RateCardManagedFields
}

var (
	_ ManagedRateCard         = (*RateCard)(nil)
	_ productcatalog.RateCard = (*RateCard)(nil)
)

type RateCard struct {
	productcatalog.RateCard
	RateCardManagedFields
}

func (r *RateCard) ManagedFields() RateCardManagedFields {
	return r.RateCardManagedFields
}

func (r *RateCard) Equal(v productcatalog.RateCard) bool {
	if managed, ok := (v).(ManagedRateCard); ok {
		if !r.RateCardManagedFields.Equal(managed.ManagedFields()) {
			return false
		}
	}

	if !r.RateCard.Equal(v) {
		return false
	}

	return true
}

func (r *RateCard) Validate() error {
	var errs []error

	if err := r.RateCard.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := r.RateCardManagedFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.AddonID == "" {
		errs = append(errs, errors.New("addonId must not be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r *RateCard) MarshalJSON() ([]byte, error) {
	serde := struct {
		productcatalog.RateCardSerde
		productcatalog.RateCard
		RateCardManagedFields
	}{
		RateCardSerde: productcatalog.RateCardSerde{
			Type: r.Type(),
		},
		RateCard:              r.RateCard,
		RateCardManagedFields: r.RateCardManagedFields,
	}

	return json.Marshal(serde)
}

func (r *RateCard) UnmarshalJSON(b []byte) error {
	var s productcatalog.RateCardSerde
	err := json.Unmarshal(b, &s)
	if err != nil {
		return fmt.Errorf("failed to JSON deserialize RateCard type: %w", err)
	}

	serde := struct {
		productcatalog.RateCard
		RateCardManagedFields
	}{
		RateCardManagedFields: r.RateCardManagedFields,
		RateCard:              r.RateCard,
	}

	switch s.Type {
	case productcatalog.FlatFeeRateCardType:
		serde.RateCard = &productcatalog.FlatFeeRateCard{}
	case productcatalog.UsageBasedRateCardType:
		serde.RateCard = &productcatalog.UsageBasedRateCard{}
	default:
		return fmt.Errorf("invalid RateCard type: %s", s.Type)
	}

	err = json.Unmarshal(b, &serde)
	if err != nil {
		return fmt.Errorf("failed to JSON deserialize UsageBasedRateCard: %w", err)
	}

	r.RateCardManagedFields = serde.RateCardManagedFields
	r.RateCard = serde.RateCard

	return nil
}

type RateCards []RateCard

func (c RateCards) At(idx int) RateCard {
	return c[idx]
}

func (c RateCards) AsProductCatalogRateCards() productcatalog.RateCards {
	var rcs productcatalog.RateCards

	for _, rc := range c {
		rcs = append(rcs, rc.RateCard)
	}

	return rcs
}

func (c RateCards) IsAligned() bool {
	periods := make(map[isodate.String]struct{})

	for _, rc := range c {
		// An effective price of 0 is still counted as a billable item
		if rc.AsMeta().Price != nil {
			// One time prices are excluded
			if d := rc.GetBillingCadence(); d != nil {
				periods[d.Normalise(true).ISOString()] = struct{}{}
			}
		}
	}

	return len(periods) <= 1
}

func (c RateCards) Equal(v RateCards) bool {
	if len(c) != len(v) {
		return false
	}

	leftSet := make(map[string]RateCard)
	for _, rc := range c {
		leftSet[rc.Key()] = rc
	}

	rightSet := make(map[string]RateCard)
	for _, rc := range v {
		rightSet[rc.Key()] = rc
	}

	if len(leftSet) != len(rightSet) {
		return false
	}

	var visited int
	for key, left := range leftSet {
		right, ok := rightSet[key]
		if !ok {
			return false
		}

		if !left.Equal(&right) {
			return false
		}

		visited++
	}

	return visited == len(rightSet)
}

func (c RateCards) Validate() error {
	var errs []error

	for _, rc := range c {
		if err := rc.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
