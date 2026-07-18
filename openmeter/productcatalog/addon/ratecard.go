package addon

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
	rateCard, err := marshalRateCardForJSON(r.RateCard)
	if err != nil {
		return nil, err
	}

	serde := struct {
		productcatalog.RateCardSerde
		RateCard json.RawMessage `json:"RateCard"`
		RateCardManagedFields
	}{
		RateCardSerde: productcatalog.RateCardSerde{
			Type: r.Type(),
		},
		RateCard:              rateCard,
		RateCardManagedFields: r.RateCardManagedFields,
	}

	return json.Marshal(serde)
}

func (r *RateCard) UnmarshalJSON(b []byte) error {
	var serialized struct {
		productcatalog.RateCardSerde
		RateCard json.RawMessage `json:"RateCard"`
		RateCardManagedFields
	}
	err := json.Unmarshal(b, &serialized)
	if err != nil {
		return fmt.Errorf("failed to JSON deserialize RateCard type: %w", err)
	}

	var rateCard productcatalog.RateCard
	switch serialized.Type {
	case productcatalog.FlatFeeRateCardType:
		rateCard = &productcatalog.FlatFeeRateCard{}
	case productcatalog.UsageBasedRateCardType:
		rateCard = &productcatalog.UsageBasedRateCard{}
	default:
		return fmt.Errorf("invalid RateCard type: %s", serialized.Type)
	}

	var currencyData struct {
		Currency *currencyx.Code `json:"currency,omitempty"`
	}
	if err = json.Unmarshal(serialized.RateCard, &currencyData); err != nil {
		return fmt.Errorf("failed to JSON deserialize RateCard currency: %w", err)
	}

	rateCardData, err := rateCardJSONWithoutCurrency(serialized.RateCard)
	if err != nil {
		return fmt.Errorf("failed to JSON deserialize RateCard: %w", err)
	}
	if err = json.Unmarshal(rateCardData, rateCard); err != nil {
		return fmt.Errorf("failed to JSON deserialize RateCard: %w", err)
	}

	if currencyData.Currency != nil {
		if err = setRateCardCurrencyIdentity(rateCard, *currencyData.Currency); err != nil {
			return fmt.Errorf("failed to set RateCard currency: %w", err)
		}
	}

	r.RateCardManagedFields = serialized.RateCardManagedFields
	r.RateCard = rateCard

	return nil
}

func marshalRateCardForJSON(rateCard productcatalog.RateCard) (json.RawMessage, error) {
	currency := rateCard.AsMeta().Currency
	rateCardWithoutCurrency := rateCard.Clone()
	if err := setRateCardCurrencyIdentity(rateCardWithoutCurrency, nil); err != nil {
		return nil, fmt.Errorf("failed to prepare RateCard for JSON serialization: %w", err)
	}

	data, err := json.Marshal(rateCardWithoutCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize RateCard: %w", err)
	}

	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("failed to rewrite RateCard currency: %w", err)
	}

	if currency != nil {
		fields["currency"], err = json.Marshal(currency.GetCode())
		if err != nil {
			return nil, fmt.Errorf("failed to JSON serialize RateCard currency: %w", err)
		}
	} else {
		delete(fields, "currency")
	}

	data, err = json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize RateCard: %w", err)
	}

	return data, nil
}

func rateCardJSONWithoutCurrency(data json.RawMessage) (json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	delete(fields, "currency")

	return json.Marshal(fields)
}

func setRateCardCurrencyIdentity(rateCard productcatalog.RateCard, currency currencyx.CurrencyIdentity) error {
	switch rateCard := rateCard.(type) {
	case *productcatalog.FlatFeeRateCard:
		rateCard.Currency = currency
	case *productcatalog.UsageBasedRateCard:
		rateCard.Currency = currency
	default:
		return fmt.Errorf("unsupported RateCard type: %T", rateCard)
	}

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

func (c RateCards) SingleBillingCadence() bool {
	return c.AsProductCatalogRateCards().SingleBillingCadence()
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
