package credit

import (
	"net/http"
)

// DELETEME_Balance of a subject in a credit.
type DELETEME_Balance struct {
	LedgerID        LedgerID                  `json:"id"`
	Metadata        map[string]string         `json:"metadata,omitempty"`
	Subject         string                    `json:"subject"`
	FeatureBalances []DELETEME_FeatureBalance `json:"featureBalances"`
	GrantBalances   []DELETEME_GrantBalance   `json:"grantBalances"`
}

// Render implements the chi renderer interface.
func (c DELETEME_Balance) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type DELETEME_GrantBalance struct {
	Grant
	Balance float64 `json:"balance"`
}

type DELETEME_FeatureBalance struct {
	Feature
	Balance float64 `json:"balance"`
	Usage   float64 `json:"usage"`
}
