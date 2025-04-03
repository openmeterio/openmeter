package subscriptionaddon

type SubscriptionAddonRateCard struct {
	RateCardID string `json:"rateCardID"` // TODO: replace with [RateCard Addon.RateCard] once exixts

	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}

type CreateSubscriptionAddonRateCardInput struct {
	RateCardID string `json:"rateCardID"`

	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}
