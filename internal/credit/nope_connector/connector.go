package nope_connector

import (
	"context"
	"fmt"
	"time"

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
func (c *Connector) GetGrant(ctx context.Context, namespace string, id string) (credit_model.Grant, error) {
	return credit_model.Grant{}, fmt.Errorf("not implemented")
}

// Credit
func (c *Connector) GetBalance(ctx context.Context, namespace string, subject string, cutline time.Time) (credit_model.Balance, error) {
	return credit_model.Balance{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHistory(ctx context.Context, namespace string, subject string, from time.Time, to time.Time, limit int) (credit_model.LedgerEntryList, error) {
	return credit_model.LedgerEntryList{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHighWatermark(ctx context.Context, namespace string, subject string) (credit_model.HighWatermark, error) {
	return credit_model.HighWatermark{}, fmt.Errorf("not implemented")
}
func (c *Connector) Reset(ctx context.Context, namespace string, reset credit_model.Reset) (credit_model.Reset, []credit_model.Grant, error) {
	return credit_model.Reset{}, []credit_model.Grant{}, fmt.Errorf("not implemented")
}

// Feature
func (c *Connector) CreateFeature(ctx context.Context, namespace string, feature credit_model.Feature) (credit_model.Feature, error) {
	return credit_model.Feature{}, fmt.Errorf("not implemented")
}
func (c *Connector) DeleteFeature(ctx context.Context, namespace string, id string) error {
	return fmt.Errorf("not implemented")
}
func (c *Connector) ListFeatures(ctx context.Context, namespace string, params credit.ListFeaturesParams) ([]credit_model.Feature, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetFeature(ctx context.Context, namespace string, id string) (credit_model.Feature, error) {
	return credit_model.Feature{}, fmt.Errorf("not implemented")
}
