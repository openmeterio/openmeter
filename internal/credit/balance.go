package credit

import (
	"net/http"
)

// Balance of a subject in a credit.
type Balance struct {
	LedgerID        LedgerID          `json:"id"`
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

type FeatureBalance struct {
	Feature
	Balance float64 `json:"balance"`
	Usage   float64 `json:"usage"`
}
