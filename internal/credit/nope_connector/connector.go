package nope_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/credit"
)

type Connector struct{}

// Implement the Connector interface
var _ credit.Connector = &Connector{}

func NewConnector() credit.Connector {
	connector := Connector{}

	return &connector
}

// Ledger
func (c *Connector) CreateLedger(ctx context.Context, namespace string, ledger credit.Ledger) (credit.Ledger, error) {
	return credit.Ledger{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListLedgers(ctx context.Context, namespace string, params credit.ListLedgersParams) ([]credit.Ledger, error) {
	return nil, fmt.Errorf("not implemented")
}

// Grant
func (c *Connector) CreateGrant(ctx context.Context, namespace string, grant credit.Grant) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) VoidGrant(ctx context.Context, namespace string, grant credit.Grant) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListGrants(ctx context.Context, namespace string, params credit.ListGrantsParams) ([]credit.Grant, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetGrant(ctx context.Context, namespace string, id ulid.ULID) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}

// Credit
func (c *Connector) GetBalance(ctx context.Context, namespace string, ledgerID ulid.ULID, cutline time.Time) (credit.Balance, error) {
	return credit.Balance{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHistory(ctx context.Context, namespace string, ledgerID ulid.ULID, from time.Time, to time.Time, limit int) (credit.LedgerEntryList, error) {
	return credit.LedgerEntryList{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHighWatermark(ctx context.Context, namespace string, ledgerID ulid.ULID) (credit.HighWatermark, error) {
	return credit.HighWatermark{}, fmt.Errorf("not implemented")
}
func (c *Connector) Reset(ctx context.Context, namespace string, reset credit.Reset) (credit.Reset, []credit.Grant, error) {
	return credit.Reset{}, []credit.Grant{}, fmt.Errorf("not implemented")
}

// Feature
func (c *Connector) CreateFeature(ctx context.Context, namespace string, feature credit.Feature) (credit.Feature, error) {
	return credit.Feature{}, fmt.Errorf("not implemented")
}
func (c *Connector) DeleteFeature(ctx context.Context, namespace string, featureID ulid.ULID) error {
	return fmt.Errorf("not implemented")
}
func (c *Connector) ListFeatures(ctx context.Context, namespace string, params credit.ListFeaturesParams) ([]credit.Feature, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetFeature(ctx context.Context, namespace string, featureID ulid.ULID) (credit.Feature, error) {
	return credit.Feature{}, fmt.Errorf("not implemented")
}
