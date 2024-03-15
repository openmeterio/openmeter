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

package booleanentitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type Connector interface {
	entitlement.SubTypeConnector
}

type connector struct{}

func NewBooleanEntitlementConnector() Connector {
	return &connector{}
}

func (c *connector) GetValue(entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	_, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	return &BooleanEntitlementValue{}, nil
}

func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature productcatalog.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	model.EntitlementType = entitlement.EntitlementTypeBoolean
	if model.MeasureUsageFrom != nil ||
		model.IssueAfterReset != nil ||
		model.IsSoftLimit != nil ||
		model.Config != nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Invalid inputs for type"}
	}

	var usagePeriod *entitlement.UsagePeriod
	var currentUsagePeriod *recurrence.Period

	if model.UsagePeriod != nil {
		usagePeriod = model.UsagePeriod

		calculatedPeriod, err := usagePeriod.GetCurrentPeriodAt(clock.Now())
		if err != nil {
			return nil, err
		}

		currentUsagePeriod = &calculatedPeriod
	}

	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:          model.Namespace,
		FeatureID:          feature.ID,
		FeatureKey:         feature.Key,
		SubjectKey:         model.SubjectKey,
		EntitlementType:    model.EntitlementType,
		Metadata:           model.Metadata,
		UsagePeriod:        model.UsagePeriod,
		CurrentUsagePeriod: currentUsagePeriod,
	}, nil
}

func (c *connector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

type BooleanEntitlementValue struct{}

var _ entitlement.EntitlementValue = &BooleanEntitlementValue{}

func (v *BooleanEntitlementValue) HasAccess() bool {
	return true
}
