package events

import (
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
)

const (
	EntitlementBalanceSnapshotV1Type = "v1.entitlements.balance.snapshot"
)

type EventMeta struct {
	Type   string
	ID     string
	Source string
	Time   time.Time
}

func (m EventMeta) NewEvent() event.Event {
	ev := event.New()

	// Required fields
	ev.SetTime(m.Time)
	ev.SetSource(m.Source)
	ev.SetID(m.ID)

	return ev
}

func CreateEntitlementsBalanceSnapshotMessage(meta EventMeta, snapshot EntitlementsBalanceSnapshotV1Properties) (event.Event, error) {
	ev := meta.NewEvent()
	ev.SetType(EntitlementBalanceSnapshotV1Type)

	ev.SetData("application/json", snapshot)

	return ev, nil
}
