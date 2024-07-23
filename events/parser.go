package events

/**

func ValidateAndParse(data []byte) (cloudevents.Event, any, error) {
}

type Payload struct {
	Snapshot *SnapType
	Notification *NotifType
}


func ValidateAndParse(data []byte) (cloudevents.Event, Payload, error) {
}

type Handler struct {
	OnSnapshot func(Snapshot *SnapType) error
	OnNotification
}

func ValidateAndParse(data []byte, handler Handler) (bool, error) {
	switch v := payload.(type) {
	case *PayloadEntitlementsBalanceNotificationV1:
		if handler.OnNotification != nil {
			return true, handler.OnNotification(v)
		}
	}

	return false, nil
}

ValidateAndParse(data []byte, Handler{
	OnSnapshot: func(event IngestEvent) {

}
})


func ParseBalanceNotification(event *event.Event) (*PayloadEntitlementsBalanceNotificationV1, error) {
}

*/
