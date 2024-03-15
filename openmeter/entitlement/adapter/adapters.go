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

package adapter

import (
	"github.com/openmeterio/openmeter/internal/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entdb"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
)

func NewPostgresEntitlementDBAdapter(db *entdb.DBClient) entitlement.EntitlementRepo {
	return adapter.NewPostgresEntitlementRepo(db)
}

func NewPostgresUsageResetDBAdapter(db *entdb.DBClient) meteredentitlement.UsageResetRepo {
	return adapter.NewPostgresUsageResetRepo(db)
}
