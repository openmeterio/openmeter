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

package memorydedupe_test

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
)

func TestIsUniqueDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")

	isUnique, err := deduplicator.IsUnique(context.Background(), namespace, ev)
	isUnique2, err2 := deduplicator.IsUnique(context.Background(), namespace, ev)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}

func TestDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	item := dedupe.Item{
		Namespace: "default",
		ID:        "id",
		Source:    "source",
	}

	isUnique, err := deduplicator.CheckUnique(context.Background(), item)
	errSet := deduplicator.Set(context.Background(), item)
	isUnique2, err2 := deduplicator.CheckUnique(context.Background(), item)
	require.NoError(t, errSet)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}
