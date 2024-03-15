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

package balanceworker

import "context"

type BalanceWorkerRepository interface {
	ListAffectedEntitlements(ctx context.Context, filterPairs []IngestEventQueryFilter) ([]IngestEventDataResponse, error)
}

type IngestEventQueryFilter struct {
	Namespace  string
	SubjectKey string
	MeterSlugs []string
}

type IngestEventDataResponse struct {
	Namespace     string
	EntitlementID string
	SubjectKey    string
	// not all entitlements have a meter associated
	MeterSlug *string
}
