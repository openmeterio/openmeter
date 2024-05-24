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
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
)

// Get credit balance, GET /api/v1/ledgers/{ledgerID}/history
func (a *Router) GetLedgerHistory(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, params api.GetLedgerHistoryParams) {
	a.CreditHandlers.GetLedgerHistory.With(creditdriver.GetLedgerHistoryRequest{
		GetLedgerHistoryParams: params,
		LedgerID:               ledgerID,
	}).ServeHTTP(w, r)
}

// CreateLedger POST /api/v1/ledgers
func (a *Router) CreateLedger(w http.ResponseWriter, r *http.Request) {
	a.CreditHandlers.CreateLedger.ServeHTTP(w, r)
}

// ListLedgers GET /api/v1/ledgers?subject=X&offset=Y&limit=Z
func (a *Router) ListLedgers(w http.ResponseWriter, r *http.Request, params api.ListLedgersParams) {
	a.CreditHandlers.ListLedgers.With(params).ServeHTTP(w, r)
}
