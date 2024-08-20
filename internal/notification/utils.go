package notification

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
