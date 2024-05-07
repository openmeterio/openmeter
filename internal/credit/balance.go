package credit

import (
	"net/http"

	"github.com/oklog/ulid/v2"
)

// Balance of a subject in a credit.
type Balance struct {
	LedgerID        ulid.ULID         `json:"id"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Subject         string            `json:"subject"`
	FeatureBalances []FeatureBalance  `json:"featureBalances"`
	GrantBalances   []GrantBalance    `json:"grantBalances"`
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

type FeatureBalance struct {
	Feature
	Balance float64 `json:"balance"`
}

// Render implements the chi renderer interface.
func (c FeatureBalance) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
