package addon

import (
	"encoding/json"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type addonAlias Addon

func currencyCodeForJSON(currency currencyx.CurrencyIdentity) currencyx.Code {
	if currency == nil {
		return ""
	}

	return currency.GetCode()
}

func (a Addon) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		addonAlias
		Currency currencyx.Code `json:"currency"`
	}{
		addonAlias: addonAlias(a),
		Currency:   currencyCodeForJSON(a.Currency),
	})
}

func (a *Addon) UnmarshalJSON(data []byte) error {
	var serialized struct {
		addonAlias
		Currency currencyx.Code `json:"currency"`
	}

	if err := json.Unmarshal(data, &serialized); err != nil {
		return err
	}

	*a = Addon(serialized.addonAlias)
	a.Currency = serialized.Currency

	return nil
}
