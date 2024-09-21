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
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) UpsertSubject(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "upsertSubject")

	err := fmt.Errorf("not implemented: subject management is only available in OpenMeter Cloud")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}

func (a *Router) GetSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getSubject")
	ctx = contextx.WithAttr(ctx, "id", idOrKey)

	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}

func (a *Router) ListSubjects(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listSubjects")
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}

func (a *Router) DeleteSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "deleteSubject")
	ctx = contextx.WithAttr(ctx, "id", idOrKey)

	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}
