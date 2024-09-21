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

package meteredentitlement

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

const (
	DefaultIssueAfterResetPriority = 1
	IssueAfterResetMetaTag         = "issueAfterReset"
)

// IssueAfterReset defines a default grant's parameters that can be created alongside an entitlement to set up a default balance.
type IssueAfterReset struct {
	Amount   float64 `json:"amount"`
	Priority *uint8  `json:"priority,omitempty"`
}

type Entitlement struct {
	entitlement.GenericProperties

	// MeasureUsageFrom defines the time from which usage should be measured.
	// This is a global value, in most cases the same value as `CreatedAt` should be fine.
	MeasureUsageFrom time.Time `json:"measureUsageFrom,omitempty"`

	// Sets up a default grant
	IssueAfterReset *IssueAfterReset `json:"issueAfterReset,omitempty"`

	// IsSoftLimit defines if the entitlement is a soft limit. By default when balance falls to 0
	// access will be disabled. If this is a soft limit, access will be allowed nonetheless.
	IsSoftLimit bool `json:"isSoftLimit,omitempty"`

	// UsagePeriod defines the recurring period for usage calculations.
	UsagePeriod entitlement.UsagePeriod `json:"usagePeriod,omitempty"`

	// CurrentPeriod defines the current period for usage calculations.
	CurrentUsagePeriod recurrence.Period `json:"currentUsagePeriod,omitempty"`

	// PreserveOverageAtReset defines if overage should be preserved when the entitlement is reset.
	PreserveOverageAtReset bool `json:"preserveOverageAtReset,omitempty"`

	// LastReset defines the last time the entitlement was reset.
	LastReset time.Time `json:"lastReset"`
}

// HasDefaultGrant returns true if the entitlement has a default grant.
// This is the case when `IssueAfterReset` is set and greater than 0.
func (e *Entitlement) HasDefaultGrant() bool {
	return e.IssueAfterReset != nil && e.IssueAfterReset.Amount > 0
}

func ParseFromGenericEntitlement(model *entitlement.Entitlement) (*Entitlement, error) {
	if model.EntitlementType != entitlement.EntitlementTypeMetered {
		return nil, &entitlement.WrongTypeError{Expected: entitlement.EntitlementTypeMetered, Actual: model.EntitlementType}
	}

	if model.MeasureUsageFrom == nil {
		return nil, &entitlement.InvalidValueError{Message: "MeasureUsageFrom is required", Type: model.EntitlementType}
	}

	if model.IsSoftLimit == nil {
		return nil, &entitlement.InvalidValueError{Message: "IsSoftLimit is required", Type: model.EntitlementType}
	}

	if model.UsagePeriod == nil {
		return nil, &entitlement.InvalidValueError{Message: "UsagePeriod is required", Type: model.EntitlementType}
	}

	if model.LastReset == nil {
		return nil, &entitlement.InvalidValueError{Message: "LastReset is required", Type: model.EntitlementType}
	}

	if model.CurrentUsagePeriod == nil {
		return nil, &entitlement.InvalidValueError{Message: "CurrentUsagePeriod is required", Type: model.EntitlementType}
	}

	if model.IssueAfterResetPriority != nil && model.IssueAfterReset == nil {
		return nil, &entitlement.InvalidValueError{Message: "IssueAfterReset is required for IssueAfterResetPriority", Type: model.EntitlementType}
	}

	ent := Entitlement{
		GenericProperties: model.GenericProperties,

		MeasureUsageFrom:       *model.MeasureUsageFrom,
		IsSoftLimit:            *model.IsSoftLimit,
		UsagePeriod:            *model.UsagePeriod,
		LastReset:              *model.LastReset,
		CurrentUsagePeriod:     *model.CurrentUsagePeriod,
		PreserveOverageAtReset: defaultx.WithDefault(model.PreserveOverageAtReset, false),
	}

	if model.IssueAfterReset != nil {
		ent.IssueAfterReset = &IssueAfterReset{
			Amount:   *model.IssueAfterReset,
			Priority: model.IssueAfterResetPriority,
		}
	}

	return &ent, nil
}
