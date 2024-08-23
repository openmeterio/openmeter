package notification

import "fmt"

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
