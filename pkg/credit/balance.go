package credit

import (
	"net/http"

	product_model "github.com/openmeterio/openmeter/pkg/product"
)

// Balance of a subject in a credit.
type Balance struct {
	Subject         string           `json:"subject"`
	ProductBalances []ProductBalance `json:"productBalances"`
	GrantBalances   []GrantBalance   `json:"grantBalances"`
}

// Render implements the chi renderer interface.
func (c Balance) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type GrantBalance struct {
	Grant
	Balance float64 `json:"balance"`
}

// Render implements the chi renderer interface.
func (c GrantBalance) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type ProductBalance struct {
	product_model.Product
	Balance float64 `json:"balance"`
}

// Render implements the chi renderer interface.
func (c ProductBalance) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
