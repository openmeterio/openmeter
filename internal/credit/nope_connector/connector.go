// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nope_connector

import (
	"context"
	"fmt"
	"time"

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
func (c *Connector) CreateLedger(ctx context.Context, ledger credit.Ledger) (credit.Ledger, error) {
	return credit.Ledger{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListLedgers(ctx context.Context, params credit.ListLedgersParams) ([]credit.Ledger, error) {
	return nil, fmt.Errorf("not implemented")
}

// Grant
func (c *Connector) CreateGrant(ctx context.Context, grant credit.Grant) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) VoidGrant(ctx context.Context, grant credit.Grant) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}
func (c *Connector) ListGrants(ctx context.Context, params credit.ListGrantsParams) ([]credit.Grant, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetGrant(ctx context.Context, grantID credit.NamespacedGrantID) (credit.Grant, error) {
	return credit.Grant{}, fmt.Errorf("not implemented")
}

// Credit
func (c *Connector) GetBalance(ctx context.Context, ledgerID credit.NamespacedLedgerID, cutline time.Time) (credit.Balance, error) {
	return credit.Balance{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHistory(ctx context.Context, ledgerID credit.NamespacedLedgerID, from time.Time, to time.Time, pagination credit.Pagination) (credit.LedgerEntryList, error) {
	return credit.LedgerEntryList{}, fmt.Errorf("not implemented")
}
func (c *Connector) GetHighWatermark(ctx context.Context, ledgerID credit.NamespacedLedgerID) (credit.HighWatermark, error) {
	return credit.HighWatermark{}, fmt.Errorf("not implemented")
}
func (c *Connector) Reset(ctx context.Context, reset credit.Reset) (credit.Reset, []credit.Grant, error) {
	return credit.Reset{}, []credit.Grant{}, fmt.Errorf("not implemented")
}

// Feature
func (c *Connector) CreateFeature(ctx context.Context, feature credit.Feature) (credit.Feature, error) {
	return credit.Feature{}, fmt.Errorf("not implemented")
}
func (c *Connector) DeleteFeature(ctx context.Context, featureID credit.NamespacedFeatureID) error {
	return fmt.Errorf("not implemented")
}
func (c *Connector) ListFeatures(ctx context.Context, params credit.ListFeaturesParams) ([]credit.Feature, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Connector) GetFeature(ctx context.Context, featureID credit.NamespacedFeatureID) (credit.Feature, error) {
	return credit.Feature{}, fmt.Errorf("not implemented")
}
