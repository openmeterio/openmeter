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

package notification

import (
	"fmt"

	"github.com/samber/lo"
)

// ChannelTypes returns a set of ChannelType from Channel slice
func ChannelTypes(channels []Channel) []ChannelType {
	seen := make(map[ChannelType]struct{}, len(channels))
	types := make([]ChannelType, 0, len(channels))

	for _, channel := range channels {
		if _, ok := seen[channel.Type]; ok {
			continue
		}

		seen[channel.Type] = struct{}{}
		types = append(types, channel.Type)
	}

	return types
}

// ChannelIDsByType returns a list of Channel identifiers from Channel slice with the provided ChannelType
func ChannelIDsByType(channels []Channel, channelType ChannelType) []string {
	ids := make([]string, 0, len(channels))

	for _, channel := range channels {
		if channel.Type != channelType {
			continue
		}

		ids = append(ids, channel.ID)
	}

	return ids
}

// DeliveryStatusStates returns a list of EventDeliveryStatusState from EventDeliveryStatus slice
func DeliveryStatusStates(statuses []EventDeliveryStatus) []EventDeliveryStatusState {
	seen := make(map[EventDeliveryStatusState]struct{}, len(statuses))
	types := make([]EventDeliveryStatusState, 0, len(statuses))

	for _, status := range statuses {
		if _, ok := seen[status.State]; ok {
			continue
		}

		seen[status.State] = struct{}{}
		types = append(types, status.State)
	}

	return types
}

func interfaceMapToStringMap(m map[string]interface{}, strict bool) (map[string]string, error) {
	var s map[string]string

	if len(m) > 0 {
		s = make(map[string]string, len(m))
		for k, v := range m {
			switch t := v.(type) {
			case string:
				s[k] = t
			case fmt.Stringer:
				s[k] = t.String()
			case int, int32, int64:
				s[k] = fmt.Sprintf("%d", t)
			case float32, float64:
				s[k] = fmt.Sprintf("%f", t)
			case bool:
				s[k] = fmt.Sprintf("%t", t)
			default:
				if strict {
					return nil, fmt.Errorf("failed to cast value with %T to string", t)
				} else {
					continue
				}
			}
		}
	}

	return s, nil
}

func StrictInterfaceMapToStringMap(m map[string]interface{}) (map[string]string, error) {
	return interfaceMapToStringMap(m, true)
}

func InterfaceMapToStringMap(m map[string]interface{}) map[string]string {
	s, _ := interfaceMapToStringMap(m, false)

	return s
}

type difference[T comparable] struct {
	leftMap map[T]struct{}
	left    []T

	rightMap map[T]struct{}
	right    []T
}

func (d difference[T]) Has(item T) bool {
	return d.HasLeft(item) || d.HasRight(item)
}

func (d difference[T]) HasLeft(item T) bool {
	if _, ok := d.leftMap[item]; ok {
		return true
	}

	return false
}

func (d difference[T]) HasRight(item T) bool {
	if _, ok := d.rightMap[item]; ok {
		return true
	}

	return false
}

func (d difference[T]) Left() []T {
	return d.left
}

func (d difference[T]) Right() []T {
	return d.right
}

func (d difference[T]) HasChanged() bool {
	return len(d.left) > 0 || len(d.right) > 0
}

func (d difference[T]) All() []T {
	return append(d.left, d.right...)
}

type ChannelIDsDifference struct {
	diff difference[string]
}

func (d ChannelIDsDifference) Has(id string) bool {
	return d.diff.Has(id)
}

func (d ChannelIDsDifference) HasChanged() bool {
	return d.diff.HasChanged()
}

func (d ChannelIDsDifference) InAdditions(id string) bool {
	return d.diff.HasLeft(id)
}

func (d ChannelIDsDifference) InRemovals(id string) bool {
	return d.diff.HasRight(id)
}

func (d ChannelIDsDifference) Additions() []string {
	return d.diff.Left()
}

func (d ChannelIDsDifference) Removals() []string {
	return d.diff.Right()
}

func (d ChannelIDsDifference) All() []string {
	return d.diff.All()
}

func NewChannelIDsDifference(new, old []string) *ChannelIDsDifference {
	left, right := lo.Difference(new, old)

	leftMap := lo.SliceToMap(left, func(item string) (string, struct{}) {
		return item, struct{}{}
	})

	rightMap := lo.SliceToMap(right, func(item string) (string, struct{}) {
		return item, struct{}{}
	})

	return &ChannelIDsDifference{
		diff: difference[string]{
			leftMap:  leftMap,
			left:     left,
			rightMap: rightMap,
			right:    right,
		},
	}
}
