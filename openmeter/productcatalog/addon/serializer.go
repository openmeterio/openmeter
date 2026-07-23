package addon

import (
	"encoding/json"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type addonAlias Addon

func currencyCodeForJSON(currency currencies.CurrencyReference) currencyx.Code {
	return currency.Code
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
	a.Currency = currencies.NewCurrencyReference(serialized.Currency)

	return nil
}
