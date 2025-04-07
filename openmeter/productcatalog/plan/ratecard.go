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
		errs = append(errs, errors.New("namespace must not be empty"))
	}

	if m.ID == "" {
		errs = append(errs, errors.New("id must not be empty"))
	}

	if m.PhaseID == "" {
		errs = append(errs, errors.New("phaseID must not be empty"))
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

	if r.PhaseID == "" {
		errs = append(errs, errors.New("phaseId must not be empty"))
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

//
//func NewRateCardFrom[T FlatFeeRateCard | UsageBasedRateCard | ~[]byte](v T) (productcatalog.RateCard, error) {
//	var rc productcatalog.RateCard
//
//	switch any(v).(type) {
//	case FlatFeeRateCard:
//		rc = lo.ToPtr(any(v).(FlatFeeRateCard))
//	case UsageBasedRateCard:
//		rc = lo.ToPtr(any(v).(UsageBasedRateCard))
//	case json.RawMessage, []byte:
//		b := any(v).([]byte)
//
//		serde := &productcatalog.RateCardSerde{}
//		if err := json.Unmarshal(b, serde); err != nil {
//			return nil, fmt.Errorf("failed to JSON deserialize RateCard type: %w", err)
//		}
//
//		switch serde.Type {
//		case productcatalog.FlatFeeRateCardType:
//			vv := FlatFeeRateCard{}
//			if err := json.Unmarshal(b, &vv); err != nil {
//				return nil, fmt.Errorf("failed to JSON deserialize FlatFeeRateCard: %w", err)
//			}
//
//			rc = &vv
//		case productcatalog.UsageBasedRateCardType:
//			vv := UsageBasedRateCard{}
//			if err := json.Unmarshal(b, &vv); err != nil {
//				return nil, fmt.Errorf("failed to JSON deserialize UsageBasedRateCard: %w", err)
//			}
//
//			rc = &vv
//		default:
//			return nil, fmt.Errorf("invalid RateCard type: %s", serde.Type)
//		}
//	}
//
//	return rc, nil
//}
//
//var _ ManagedRateCard = (*FlatFeeRateCard)(nil)
//
//type FlatFeeRateCard struct {
//	RateCardManagedFields
//	productcatalog.FlatFeeRateCard
//}
//
//func (r *FlatFeeRateCard) ManagedFields() RateCardManagedFields {
//	return r.RateCardManagedFields
//}
//
//func (r *FlatFeeRateCard) MarshalJSON() ([]byte, error) {
//	type flatFeeRateCardSerde FlatFeeRateCard
//	serde := struct {
//		productcatalog.RateCardSerde
//		flatFeeRateCardSerde
//	}{
//		RateCardSerde: productcatalog.RateCardSerde{
//			Type: productcatalog.FlatFeeRateCardType,
//		},
//		flatFeeRateCardSerde: flatFeeRateCardSerde(*r),
//	}
//
//	return json.Marshal(serde)
//}
//
//func (r *FlatFeeRateCard) UnmarshalJSON(b []byte) error {
//	serde := struct {
//		productcatalog.RateCardSerde
//		RateCardManagedFields
//		productcatalog.FlatFeeRateCard
//	}{
//		RateCardSerde: productcatalog.RateCardSerde{
//			Type: productcatalog.FlatFeeRateCardType,
//		},
//		RateCardManagedFields: r.RateCardManagedFields,
//		FlatFeeRateCard:       r.FlatFeeRateCard,
//	}
//
//	err := json.Unmarshal(b, &serde)
//	if err != nil {
//		return fmt.Errorf("failed to JSON deserialize FlatFeeRateCard: %w", err)
//	}
//
//	r.RateCardManagedFields = serde.RateCardManagedFields
//	r.FlatFeeRateCard = serde.FlatFeeRateCard
//
//	return nil
//}
//
//func (r *FlatFeeRateCard) Equal(v productcatalog.RateCard) bool {
//	switch vv := v.(type) {
//	case *FlatFeeRateCard:
//		if !r.RateCardManagedFields.Equal(vv.RateCardManagedFields) {
//			return false
//		}
//
//		if r.PhaseID != vv.PhaseID {
//			return false
//		}
//
//		return r.FlatFeeRateCard.Equal(&vv.FlatFeeRateCard)
//	case *productcatalog.UsageBasedRateCard:
//		return r.FlatFeeRateCard.Equal(vv)
//	default:
//		return false
//	}
//}
//
//func (r *FlatFeeRateCard) Validate() error {
//	var errs []error
//
//	if err := r.FlatFeeRateCard.Validate(); err != nil {
//		errs = append(errs, err)
//	}
//
//	return models.NewNillableGenericValidationError(errors.Join(errs...))
//}
//
//func (r *FlatFeeRateCard) Merge(v productcatalog.RateCard) error {
//	switch vv := v.(type) {
//	case *FlatFeeRateCard:
//		err := r.FlatFeeRateCard.Merge(&vv.FlatFeeRateCard)
//		if err != nil {
//			return err
//		}
//	case *productcatalog.FlatFeeRateCard:
//		err := r.FlatFeeRateCard.Merge(vv)
//		if err != nil {
//			return err
//		}
//	default:
//		return fmt.Errorf("invalid FlatFeeRateCard type: %T", vv)
//	}
//
//	return nil
//}
//
//var _ ManagedRateCard = (*UsageBasedRateCard)(nil)
//
//type UsageBasedRateCard struct {
//	RateCardManagedFields
//	productcatalog.UsageBasedRateCard
//}
//
//func (r *UsageBasedRateCard) ManagedFields() RateCardManagedFields {
//	return r.RateCardManagedFields
//}
//
//func (r *UsageBasedRateCard) MarshalJSON() ([]byte, error) {
//	serde := struct {
//		productcatalog.RateCardSerde
//		RateCardManagedFields
//		productcatalog.UsageBasedRateCard
//	}{
//		RateCardSerde: productcatalog.RateCardSerde{
//			Type: productcatalog.UsageBasedRateCardType,
//		},
//		RateCardManagedFields: r.RateCardManagedFields,
//		UsageBasedRateCard:    r.UsageBasedRateCard,
//	}
//
//	return json.Marshal(serde)
//}
//
//func (r *UsageBasedRateCard) UnmarshalJSON(b []byte) error {
//	serde := struct {
//		productcatalog.RateCardSerde
//		RateCardManagedFields
//		productcatalog.UsageBasedRateCard
//	}{
//		RateCardSerde: productcatalog.RateCardSerde{
//			Type: productcatalog.UsageBasedRateCardType,
//		},
//		RateCardManagedFields: r.RateCardManagedFields,
//		UsageBasedRateCard:    r.UsageBasedRateCard,
//	}
//
//	err := json.Unmarshal(b, &serde)
//	if err != nil {
//		return fmt.Errorf("failed to JSON deserialize UsageBasedRateCard: %w", err)
//	}
//
//	r.RateCardManagedFields = serde.RateCardManagedFields
//	r.UsageBasedRateCard = serde.UsageBasedRateCard
//
//	return nil
//}
//
//func (r *UsageBasedRateCard) Equal(v productcatalog.RateCard) bool {
//	switch vv := v.(type) {
//	case *UsageBasedRateCard:
//		if !r.RateCardManagedFields.Equal(vv.RateCardManagedFields) {
//			return false
//		}
//
//		if r.PhaseID != vv.PhaseID {
//			return false
//		}
//
//		return r.UsageBasedRateCard.Equal(&vv.UsageBasedRateCard)
//	case *productcatalog.UsageBasedRateCard:
//		return r.UsageBasedRateCard.Equal(vv)
//	default:
//		return false
//	}
//}
//
//func (r *UsageBasedRateCard) Validate() error {
//	var errs []error
//
//	if err := r.UsageBasedRateCard.Validate(); err != nil {
//		errs = append(errs, err)
//	}
//
//	return models.NewNillableGenericValidationError(errors.Join(errs...))
//}
//
//func (r *UsageBasedRateCard) Merge(v productcatalog.RateCard) error {
//	switch vv := v.(type) {
//	case *UsageBasedRateCard:
//		err := r.UsageBasedRateCard.Merge(&vv.UsageBasedRateCard)
//		if err != nil {
//			return err
//		}
//	case *productcatalog.UsageBasedRateCard:
//		err := r.UsageBasedRateCard.Merge(vv)
//		if err != nil {
//			return err
//		}
//	default:
//		return fmt.Errorf("invalid UsageBasedRateCard type: %T", vv)
//	}
//
//	return nil
//}
