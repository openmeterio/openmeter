package nope_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/internal/credit"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
)

type Connector struct{}

// Implement the Connector interface
var _ credit.Connector = &Connector{}

func NewConnector() credit.Connector {
	connector := Connector{}

	return &connector
}

// Ledger
func (c *Connector) CreateLedger(ctx context.Context, namespace string, ledger credit_model.Ledger, upsert bool) (credit_model.Ledger, error) {
	return credit_model.Ledger{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListLedgers(ctx context.Context, namespace string, params credit_model.ListLedgersParams) ([]credit_model.Ledger, error) {
	return nil, fmt.Errorf("not implemented")
}

// Grant
func (c *Connector) CreateGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error) {
	return credit_model.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) VoidGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error) {
	return credit_model.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListGrants(ctx context.Context, namespace string, params credit.ListGrantsParams) ([]credit_model.Grant, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetGrant(ctx context.Context, namespace string, id ulid.ULID) (credit_model.Grant, error) {
	return credit_model.Grant{}, fmt.Errorf("not implemented")
}

// Credit
func (c *Connector) GetBalance(ctx context.Context, namespace string, ledgerID ulid.ULID, cutline time.Time) (credit_model.Balance, error) {
	return credit_model.Balance{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHistory(ctx context.Context, namespace string, ledgerID ulid.ULID, from time.Time, to time.Time, limit int) (credit_model.LedgerEntryList, error) {
	return credit_model.LedgerEntryList{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHighWatermark(ctx context.Context, namespace string, ledgerID ulid.ULID) (credit_model.HighWatermark, error) {
	return credit_model.HighWatermark{}, fmt.Errorf("not implemented")
}
func (c *Connector) Reset(ctx context.Context, namespace string, reset credit_model.Reset) (credit_model.Reset, []credit_model.Grant, error) {
	return credit_model.Reset{}, []credit_model.Grant{}, fmt.Errorf("not implemented")
}

// Feature
func (c *Connector) CreateFeature(ctx context.Context, namespace string, feature credit_model.Feature) (credit_model.Feature, error) {
	return credit_model.Feature{}, fmt.Errorf("not implemented")
}
func (c *Connector) DeleteFeature(ctx context.Context, namespace string, featureID ulid.ULID) error {
	return fmt.Errorf("not implemented")
}
func (c *Connector) ListFeatures(ctx context.Context, namespace string, params credit.ListFeaturesParams) ([]credit_model.Feature, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetFeature(ctx context.Context, namespace string, featureID ulid.ULID) (credit_model.Feature, error) {
	return credit_model.Feature{}, fmt.Errorf("not implemented")
}
