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

package entitlement

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AlreadyExistsError struct {
	EntitlementID string
	FeatureID     string
	SubjectKey    string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement with id %s already exists for feature %s and subject %s", e.EntitlementID, e.FeatureID, e.SubjectKey)
}

type NotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type WrongTypeError struct {
	Expected EntitlementType
	Actual   EntitlementType
}

func (e *WrongTypeError) Error() string {
	return fmt.Sprintf("expected entitlement type %s but got %s", e.Expected, e.Actual)
}

type InvalidValueError struct {
	Message string
	Type    EntitlementType
}

func (e *InvalidValueError) Error() string {
	return fmt.Sprintf("invalid entitlement value for type %s: %s", e.Type, e.Message)
}

type InvalidFeatureError struct {
	FeatureID string
	Message   string
}

func (e *InvalidFeatureError) Error() string {
	return fmt.Sprintf("invalid feature %s: %s", e.FeatureID, e.Message)
}
