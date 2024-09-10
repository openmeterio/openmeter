package subscriptionitem

type creatable interface {
	EntitlementCreator
}

type Connector interface {
	CreateFromRateCard(rateCard creatable) (Item, error)
}

type connector struct{}
