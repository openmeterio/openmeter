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

package ingest_test

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

func TestInMemoryCollector(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")

	err := collector.Ingest(context.Background(), namespace, ev)
	require.NoError(t, err)

	assert.Equal(t, []string{namespace}, collector.Namespaces())
	assert.Equal(t, []event.Event{ev}, collector.Events(namespace))
}
