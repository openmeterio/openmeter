package plan

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator                      = (*RateCardManagedFields)(nil)
	_ models.Equaler[RateCardManagedFields] = (*RateCardManagedFields)(nil)
)

type RateCardManagedFields struct {
	models.ManagedModel
	models.NamespacedID

	// PhaseID
	PhaseID string `json:"phaseId"`
}

func (m RateCardManagedFields) Equal(v RateCardManagedFields) bool {
	if m.Namespace != v.Namespace {
		return false
	}

	if m.ID != v.ID {
		return false
	}

	return m.PhaseID == v.PhaseID
}

func (m RateCardManagedFields) Validate() error {
	var errs []error

	if m.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if m.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	if m.PhaseID == "" {
		errs = append(errs, errors.New("managed ratecard must have plan phase reference set"))
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
