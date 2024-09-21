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

package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	creditdriver "github.com/openmeterio/openmeter/openmeter/credit/driver"
)

// List grants
// (GET /api/v1/grants)
func (a *Router) ListGrants(w http.ResponseWriter, r *http.Request, params api.ListGrantsParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ListGrants(w, r, params)
		return
	}
	a.creditHandler.ListGrants().With(creditdriver.ListGrantsHandlerParams{
		Params: params,
	}).ServeHTTP(w, r)
}

// Delete a grant
// (DELETE /api/v1/grants/{grantId})
func (a *Router) VoidGrant(w http.ResponseWriter, r *http.Request, grantId api.GrantId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.VoidGrant(w, r, grantId)
		return
	}
	a.creditHandler.VoidGrant().With(creditdriver.VoidGrantHandlerParams{
		ID: grantId,
	}).ServeHTTP(w, r)
}
