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

package credit

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

type LedgerAlreadyExistsProblemResponse struct {
	*models.StatusProblem
	ConflictingEntity Ledger `json:"conflictingEntity"`
}

func (p *LedgerAlreadyExistsProblemResponse) Respond(w http.ResponseWriter) {
	models.RespondProblem(p, w)
}

func NewLedgerAlreadyExistsProblem(ctx context.Context, err error, existingEntry Ledger) models.Problem {
	return &LedgerAlreadyExistsProblemResponse{
		StatusProblem:     models.NewStatusProblem(ctx, err, http.StatusConflict),
		ConflictingEntity: existingEntry,
	}
}
