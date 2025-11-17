package eventhandler

import (
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

func filterActiveDeliveryStatusesByChannelType(event *notification.Event, channelType notification.ChannelType) []notification.EventDeliveryStatus {
	if event == nil {
		return nil
	}

	channelIDByType := func() map[string]notification.ChannelType {
		if event.Rule.Channels == nil {
			return nil
		}

		m := make(map[string]notification.ChannelType)

		for _, channel := range event.Rule.Channels {
			m[channel.ID] = channel.Type
		}

		return m
	}()

	var result []notification.EventDeliveryStatus

	for _, deliveryStatus := range event.DeliveryStatus {
		if lo.Contains([]notification.EventDeliveryStatusState{
			notification.EventDeliveryStatusStateFailed,
			notification.EventDeliveryStatusStateSuccess,
		}, deliveryStatus.State) {
			continue
		}

		if deliveryStatus.ChannelID == "" {
			continue
		}

		chType, ok := channelIDByType[deliveryStatus.ChannelID]
		if !ok || chType != channelType {
			continue
		}

		result = append(result, deliveryStatus)
	}

	return result
}

func sortDeliveryStatusStateByPriority(states []notification.EventDeliveryStatus) []notification.EventDeliveryStatus {
	if len(states) == 0 {
		return nil
	}

	priority := map[notification.EventDeliveryStatusState]int8{
		notification.EventDeliveryStatusStatePending:   0,
		notification.EventDeliveryStatusStateResending: 1,
		notification.EventDeliveryStatusStateSending:   2,
		notification.EventDeliveryStatusStateFailed:    3,
		notification.EventDeliveryStatusStateSuccess:   3,
	}

	slices.SortFunc(states, func(a, b notification.EventDeliveryStatus) int {
		if a.State == b.State {
			return 0
		}

		if priority[a.State] < priority[b.State] {
			return -1
		}

		return 1
	})

	return states
}
