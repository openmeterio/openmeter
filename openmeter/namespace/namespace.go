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

// Package namespace adds a concept of tenancy to OpenMeter allowing to segment clients.
package namespace

import (
	"github.com/openmeterio/openmeter/internal/namespace"
)

// Manager is responsible for managing namespaces in different components.
type Manager = namespace.Manager

type ManagerConfig = namespace.ManagerConfig

func NewManager(config ManagerConfig) (*Manager, error) {
	return namespace.NewManager(config)
}

// Handler is responsible for creating a namespace in a given component.
//
// An empty name means a default namespace is supposed to be created.
// The concept of a default namespace is implementation specific.
//
// The behavior for trying to create a namespace that already exists is unspecified at the moment.
type Handler = namespace.Handler
